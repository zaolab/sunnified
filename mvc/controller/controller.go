package controller

import (
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/mvc/view"
	"github.com/zaolab/sunnified/web"
)

var ErrControllerNotFound = errors.New("controller not found")
var ErrUnprepared = errors.New("controller has not been prep'ed")
var ErrUnexecuted = errors.New("controller has not been executed")
var ErrParseStruct = errors.New("Sunnified Parser error")

const StructValueFeedTag = "sunnified.feed"
const StructValueResTag = "sunnified.res"

type StructValueFeeder interface {
	FeedStructValue(*web.Context, *FieldMeta, reflect.Value) (reflect.Value, error)
}

type ControlHandler interface {
	GetControlManager(*web.Context) *ControlManager
}

func NewControlManager(context *web.Context, cm *Meta, action string) *ControlManager {
	rtype := cm.RType()
	if rtype.Kind() == reflect.Ptr {
		rtype = rtype.Elem()
	}
	return &ControlManager{
		control:     reflect.New(rtype),
		context:     context,
		controlmeta: cm,
		action:      action,
	}
}

type ControlManager struct {
	control     reflect.Value
	context     *web.Context
	controlmeta *Meta
	action      string
	prepared    bool
	executed    bool
	state       int
	vw          mvc.View
}

func (c *ControlManager) SetControllerMeta(cm *Meta) (ok bool) {
	if !c.prepared {
		rtype := cm.RType()
		if rtype.Kind() == reflect.Ptr {
			rtype = rtype.Elem()
		}

		c.controlmeta = cm
		c.control = reflect.New(rtype)
		ok = true
	}

	return
}

func (c *ControlManager) SetAction(action string) (ok bool) {
	if !c.prepared {
		c.action = action
		ok = true
	}

	return
}

func (c *ControlManager) SetState(state int) {
	c.state = state
}

func (c *ControlManager) State() int {
	return c.state
}

func (c *ControlManager) View() mvc.View {
	return c.vw
}

func (c *ControlManager) IsPrepared() bool {
	return c.prepared
}

func (c *ControlManager) IsExecuted() bool {
	return c.executed
}

func (c *ControlManager) MvcMeta() mvc.Meta {
	if c.controlmeta != nil {
		return mvc.Meta{c.controlmeta.Module(), c.controlmeta.Name(), c.action, c.context.Ext}
	}
	return mvc.Meta{}
}

func (c *ControlManager) ModuleName() string {
	if c.controlmeta != nil {
		return c.controlmeta.Module()
	}
	return ""
}

func (c *ControlManager) ControllerName() string {
	if c.controlmeta != nil {
		return c.controlmeta.Name()
	}
	return ""
}

func (c *ControlManager) ActionName() string {
	return c.action
}

func (c *ControlManager) Controller() reflect.Value {
	return c.control
}

func (c *ControlManager) ActionMeta() *ActionMeta {
	return c.controlmeta.ActionFromRequest(c.MvcMeta()[mvc.MVCAction], c.context)
}

func (c *ControlManager) AvailableMethods() ReqMethod {
	return c.controlmeta.ActionAvailableMethods(c.action)
}

func (c *ControlManager) AvailableMethodsList() []string {
	return c.controlmeta.ActionAvailableMethodsList(c.action)
}

func (c *ControlManager) ControllerMeta() *Meta {
	return c.controlmeta
}

func (c *ControlManager) Context() *web.Context {
	return c.context
}

func (c *ControlManager) PrepareAndExecute() (state int, vw mvc.View) {
	if c.Prepare() == nil {
		return c.Execute()
	}
	return c.state, nil
}

func (c *ControlManager) Prepare() error {
	if !c.prepared && (c.state == 0 || (c.state >= 200 && c.state < 300)) {
		if c.controlmeta == nil {
			c.state = 404
			return ErrControllerNotFound
		}

		switch c.controlmeta.T() {
		case ContypeConstructor:
			results := c.control.Call(getArgSlice(c.controlmeta.Args(),
				getVMap(c.context),
				c.context.PData))
			c.control = results[0]

			if c.control.Kind() == reflect.Interface {
				c.control = c.control.Elem()
			}
			// after Elem from Interface, it might be a pointer to a struct too
			if c.control.Kind() == reflect.Ptr {
				c.control = c.control.Elem()
			}

			if c.controlmeta.Status() {
				state := int(results[1].Int())

				if state <= 0 {
					state = http.StatusOK
				}

				c.state = state
			}
		case ContypeStruct, ContypeScontroller:
			fields := c.controlmeta.Fields()
			tmpcontrol := reflect.Indirect(c.control)

			for _, field := range fields {
				value := getDataValue(&field.DataMeta,
					getVMap(c.context),
					c.context.PData)

				// allows middleware resources to make changes to value based on tag
				// this can be useful to csrf where non csrf verified values are filtered
				if res := field.Tag().Get(StructValueFeedTag); res != "" {
					var reses []string

					if strings.Contains(res, ",") {
						reses = strings.Split(res, ",")
					} else {
						reses = []string{res}
					}

					for _, r := range reses {
						rinterface := c.context.Resource(strings.TrimSpace(r))

						if rinterface != nil {
							if parser, ok := rinterface.(StructValueFeeder); ok {
								var err error
								value, err = parser.FeedStructValue(c.context, field, value)
								if err != nil {
									c.state = 500
									log.Println(err)
									return ErrParseStruct
								}
							}
						} else {
							log.Println("Resource to parse struct var not found: ", r)
							c.state = 500
							return ErrParseStruct
						}
					}
				} else if res := field.Tag().Get(StructValueResTag); res != "" {
					rinterface := c.context.Resource(strings.TrimSpace(res))

					if rinterface != nil {
						value = reflect.ValueOf(rinterface)
					}
				}

				if value.IsValid() {
					tmpcontrol.FieldByName(field.Name()).Set(value)
				}
			}

			if c.state != 500 && c.controlmeta.T() == ContypeScontroller {
				ctrler := c.control.Interface().(mvc.Controller)
				ctrler.Construct_(c.context)
			}
		}

		if c.context.HasErrorCode() {
			c.state = c.context.ErrorCode()
		}

		c.prepared = true
	}

	return nil
}

func (c *ControlManager) Execute() (state int, vw mvc.View) {
	if c.prepared {
		if c.state == 0 {
			c.state = 200
		}

		var results []reflect.Value
		var rstyle = c.controlmeta.ResultStyle

		if c.state >= http.StatusOK && c.state < http.StatusMultipleChoices {
			switch c.controlmeta.T() {
			case ContypeFunc:
				results = c.control.Call(getArgSlice(c.controlmeta.Args(),
					getVMap(c.context),
					c.context.PData))
			default:
				actmeta := c.ActionMeta()
				if actmeta != nil {
					meth := c.control.MethodByName(actmeta.Name())
					results = meth.Call(getArgSlice(actmeta.Args(),
						getVMap(c.context),
						c.context.PData))
					rstyle = actmeta.ResultStyle
				} else {
					c.state = 404
					state = c.state
					return
				}
			}
		}

		if rstyle.Status() {
			c.state = int(results[1].Int())
		}

		if rstyle.View() || rstyle.Vmap() || rstyle.MapSI() {
			// for a consistent error page, error should be returned instead and allow sunny server itself
			// to render the error page
			state = c.state

			if state == 200 || state == 0 {
				if rstyle.View() {
					if !results[0].IsNil() && results[0].IsValid() {
						c.vw = (results[0].Interface()).(mvc.View)
					}
				} else {
					var vmap mvc.VM

					if results[0].IsNil() || !results[0].IsValid() {
						vmap = mvc.VM{}
					} else if rstyle.Vmap() {
						vmap = results[0].Interface().(mvc.VM)
					} else {
						vmap = mvc.VM(results[0].Interface().(map[string]interface{}))
					}

					c.vw = view.NewResultView(vmap)
				}

				vw = c.vw
				if vw == nil {
					state = -1
				}
			}
		} else {
			// if state returned is -1, it means the controller has handled the response
			state = -1
		}

		c.executed = true
	} else {
		state = c.state
	}

	return
}

func (c *ControlManager) PublishView() (err error) {
	if !c.prepared {
		err = ErrUnprepared
	} else if !c.executed {
		err = ErrUnexecuted
	} else if c.vw != nil {
		if c.context.Request.Method == "HEAD" {
			c.context.Response.Header().Set("Content-Type", c.vw.ContentType(c.context))
		} else {
			err = c.vw.Publish(c.context)
		}
	}
	return
}

func (c *ControlManager) Cleanup() {
	if c.prepared && c.controlmeta.T() == ContypeScontroller {
		ctrler := c.control.Interface().(mvc.Controller)
		ctrler.Destruct_()
	}
}

func getVMap(context *web.Context) map[string]reflect.Value {
	return map[string]reflect.Value{
		"context":     reflect.ValueOf(context),
		"w":           reflect.ValueOf(context.Response),
		"r":           reflect.ValueOf(context.Request),
		"upath":       reflect.ValueOf(context.UPath),
		"pdata":       reflect.ValueOf(context.PData),
		"upath_slice": reflect.ValueOf([]string(context.UPath)),
		"pdata_map":   reflect.ValueOf(map[string]string(context.PData)),
	}
}

func getArgSlice(args []*ArgMeta, vmap map[string]reflect.Value, d web.PData) (values []reflect.Value) {
	values = make([]reflect.Value, len(args))

	for i, arg := range args {
		values[i] = getDataValue(&arg.DataMeta, vmap, d)
	}

	return
}

func getDataValue(arg *DataMeta, vmap map[string]reflect.Value, d web.PData) (value reflect.Value) {
	switch arg.T() {
	case DatatypeWebContext:
		value = vmap["context"]
	case DatatypeRequest:
		value = vmap["r"]
	case DatatypeResponseWriter:
		value = vmap["w"]
	case DatatypeUpath:
		value = vmap["upath"]
	case DatatypeUpathSlice:
		value = vmap["upath_slice"]
	case DatatypePdata:
		value = vmap["pdata"]
	case DatatypePdataMap:
		value = vmap["pdata_map"]
	case DatatypeString:
		val, _ := d.String(arg.LName())
		value = reflect.ValueOf(val)
	case DatatypeInt:
		val, _ := d.Int(arg.LName())
		value = reflect.ValueOf(val)
	case DatatypeInt64:
		val, _ := d.Int64(arg.LName())
		value = reflect.ValueOf(val)
	case DatatypeFloat:
		val, _ := d.Float32(arg.LName())
		value = reflect.ValueOf(val)
	case DatatypeFloat64:
		val, _ := d.Float64(arg.LName())
		value = reflect.ValueOf(val)
	case DatatypeEmail:
		val, _ := d.Email(arg.LName())
		value = reflect.ValueOf(val)
	case DatatypeURL:
		val, _ := d.Url(arg.LName())
		value = reflect.ValueOf(val)
	case DatatypeDate:
		val, _ := d.Date(arg.LName())
		value = reflect.ValueOf(val)
	case DatatypeTime:
		val, _ := d.Time(arg.LName())
		value = reflect.ValueOf(val)
	case DatatypeDateTime:
		val, _ := d.DateTime(arg.LName())
		value = reflect.ValueOf(val)
	case DatatypeStruct, DatatypeEmbedded:
		fields := arg.Fields()
		model := reflect.New(arg.RType())
		modelval := model.Elem()
		for _, field := range fields {
			modelval.FieldByName(field.Name()).Set(getDataValue(&field.DataMeta, vmap, d))
		}
		if arg.RType().Kind() == reflect.Ptr {
			value = model
		} else {
			value = modelval
		}
	}

	return
}
