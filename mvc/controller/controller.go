package controller

import (
	"errors"
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/mvc/view"
	"github.com/zaolab/sunnified/web"
	"log"
	"net/http"
	"reflect"
	"strings"
)

var ErrControllerNotFound = errors.New("Controller not found")
var ErrUnprepared = errors.New("Controller has not been prep'ed")
var ErrUnexecuted = errors.New("Controller has not been executed")
var ErrParseStruct = errors.New("Sunnified Parser error")

const STRUCTVALUEFEED_TAG = "sunnified.feed"
const STRUCTVALUERES_TAG = "sunnified.res"

type StructValueFeeder interface {
	FeedStructValue(*web.Context, *FieldMeta, reflect.Value) (reflect.Value, error)
}

type ControlHandler interface {
	GetControlManager(*web.Context) *ControlManager
}

func NewControlManager(context *web.Context, cm *ControllerMeta, action string) *ControlManager {
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
	controlmeta *ControllerMeta
	action      string
	prepared    bool
	executed    bool
	state       int
	vw          mvc.View
}

func (this *ControlManager) SetControllerMeta(cm *ControllerMeta) (ok bool) {
	if !this.prepared {
		rtype := cm.RType()
		if rtype.Kind() == reflect.Ptr {
			rtype = rtype.Elem()
		}

		this.controlmeta = cm
		this.control = reflect.New(rtype)
		ok = true
	}

	return
}

func (this *ControlManager) SetAction(action string) (ok bool) {
	if !this.prepared {
		this.action = action
		ok = true
	}

	return
}

func (this *ControlManager) SetState(state int) {
	this.state = state
}

func (this *ControlManager) State() int {
	return this.state
}

func (this *ControlManager) View() mvc.View {
	return this.vw
}

func (this *ControlManager) IsPrepared() bool {
	return this.prepared
}

func (this *ControlManager) IsExecuted() bool {
	return this.executed
}

func (this *ControlManager) MvcMeta() mvc.MvcMeta {
	if this.controlmeta != nil {
		return mvc.MvcMeta{this.controlmeta.Module(), this.controlmeta.Name(), this.action, this.context.Ext}
	}
	return mvc.MvcMeta{}
}

func (this *ControlManager) ModuleName() string {
	if this.controlmeta != nil {
		return this.controlmeta.Module()
	}
	return ""
}

func (this *ControlManager) ControllerName() string {
	if this.controlmeta != nil {
		return this.controlmeta.Name()
	}
	return ""
}

func (this *ControlManager) ActionName() string {
	return this.action
}

func (this *ControlManager) Controller() reflect.Value {
	return this.control
}

func (this *ControlManager) ActionMeta() *ActionMeta {
	return this.controlmeta.ActionFromRequest(this.MvcMeta()[mvc.MVC_ACTION], this.context)
}

func (this *ControlManager) AvailableMethods() ReqMethod {
	return this.controlmeta.ActionAvailableMethods(this.action)
}

func (this *ControlManager) AvailableMethodsList() []string {
	return this.controlmeta.ActionAvailableMethodsList(this.action)
}

func (this *ControlManager) ControllerMeta() *ControllerMeta {
	return this.controlmeta
}

func (this *ControlManager) Context() *web.Context {
	return this.context
}

func (this *ControlManager) PrepareAndExecute() (state int, vw mvc.View) {
	if this.Prepare() == nil {
		return this.Execute()
	}
	return this.state, nil
}

func (this *ControlManager) Prepare() error {
	if !this.prepared && (this.state == 0 || (this.state >= 200 && this.state < 300)) {
		if this.controlmeta == nil {
			this.state = 404
			return ErrControllerNotFound
		}

		switch this.controlmeta.T() {
		case CONTYPE_CONSTRUCTOR:
			results := this.control.Call(getArgSlice(this.controlmeta.Args(),
				getVMap(this.context),
				this.context.PData))
			this.control = results[0]

			if this.control.Kind() == reflect.Interface {
				this.control = this.control.Elem()
			}
			// after Elem from Interface, it might be a pointer to a struct too
			if this.control.Kind() == reflect.Ptr {
				this.control = this.control.Elem()
			}

			if this.controlmeta.Status() {
				state := int(results[1].Int())

				if state <= 0 {
					state = http.StatusOK
				}

				this.state = state
			}
		case CONTYPE_STRUCT:
			fallthrough
		case CONTYPE_SCONTROLLER:
			fields := this.controlmeta.Fields()
			tmpcontrol := reflect.Indirect(this.control)

			for _, field := range fields {
				value := getDataValue(&field.DataMeta,
					getVMap(this.context),
					this.context.PData)

				// allows middleware resources to make changes to value based on tag
				// this can be useful to csrf where non csrf verified values are filtered
				if res := field.Tag().Get(STRUCTVALUEFEED_TAG); res != "" {
					var reses []string

					if strings.Contains(res, ",") {
						reses = strings.Split(res, ",")
					} else {
						reses = []string{res}
					}

					for _, r := range reses {
						rinterface := this.context.Resource(strings.TrimSpace(r))

						if rinterface != nil {
							if parser, ok := rinterface.(StructValueFeeder); ok {
								var err error
								value, err = parser.FeedStructValue(this.context, field, value)
								if err != nil {
									this.state = 500
									log.Println(err)
									return ErrParseStruct
								}
							}
						} else {
							log.Println("Resource to parse struct var not found: ", r)
							this.state = 500
							return ErrParseStruct
						}
					}
				} else if res := field.Tag().Get(STRUCTVALUERES_TAG); res != "" {
					rinterface := this.context.Resource(strings.TrimSpace(res))

					if rinterface != nil {
						value = reflect.ValueOf(rinterface)
					}
				}

				if value.IsValid() {
					tmpcontrol.FieldByName(field.Name()).Set(value)
				}
			}

			if this.state != 500 && this.controlmeta.T() == CONTYPE_SCONTROLLER {
				ctrler := this.control.Interface().(mvc.Controller)
				ctrler.Construct_(this.context)
			}
		}

		this.prepared = true
	}

	return nil
}

func (this *ControlManager) Execute() (state int, vw mvc.View) {
	if this.prepared {
		if this.state == 0 {
			this.state = 200
		}

		var results []reflect.Value
		var rstyle ResultStyle = this.controlmeta.ResultStyle

		if this.state >= http.StatusOK && this.state < http.StatusMultipleChoices {
			switch this.controlmeta.T() {
			case CONTYPE_FUNC:
				results = this.control.Call(getArgSlice(this.controlmeta.Args(),
					getVMap(this.context),
					this.context.PData))
			default:
				actmeta := this.ActionMeta()
				if actmeta != nil {
					meth := this.control.MethodByName(actmeta.Name())
					results = meth.Call(getArgSlice(actmeta.Args(),
						getVMap(this.context),
						this.context.PData))
					rstyle = actmeta.ResultStyle
				} else {
					this.state = 404
					state = this.state
					return
				}
			}
		}

		if rstyle.Status() {
			this.state = int(results[1].Int())
		}

		if rstyle.View() || rstyle.Vmap() || rstyle.MapSI() {
			// for a consistent error page, error should be returned instead and allow sunny server itself
			// to render the error page
			state = this.state

			if state == 200 || state == 0 {
				if rstyle.View() {
					if !results[0].IsNil() && results[0].IsValid() {
						this.vw = (results[0].Interface()).(mvc.View)
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

					this.vw = view.NewResultView(vmap)
				}

				vw = this.vw
				if vw == nil {
					state = -1
				}
			}
		} else {
			// if state returned is -1, it means the controller has handled the response
			state = -1
		}

		this.executed = true
	} else {
		state = this.state
	}

	return
}

func (this *ControlManager) PublishView() (err error) {
	if !this.prepared {
		err = ErrUnprepared
	} else if !this.executed {
		err = ErrUnexecuted
	} else if this.vw != nil {
		if this.context.Request.Method == "HEAD" {
			this.context.Response.Header().Set("Content-Type", this.vw.ContentType(this.context))
		} else {
			err = this.vw.Publish(this.context)
		}
	}
	return
}

func (this *ControlManager) Cleanup() {
	if this.prepared && this.controlmeta.T() == CONTYPE_SCONTROLLER {
		ctrler := this.control.Interface().(mvc.Controller)
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
	case DATATYPE_WEBCONTEXT:
		value = vmap["context"]
	case DATATYPE_REQUEST:
		value = vmap["r"]
	case DATATYPE_RESPONSEWRITER:
		value = vmap["w"]
	case DATATYPE_UPATH:
		value = vmap["upath"]
	case DATATYPE_UPATH_SLICE:
		value = vmap["upath_slice"]
	case DATATYPE_PDATA:
		value = vmap["pdata"]
	case DATATYPE_PDATA_MAP:
		value = vmap["pdata_map"]
	case DATATYPE_STRING:
		val, _ := d.String(arg.LName())
		value = reflect.ValueOf(val)
	case DATATYPE_INT:
		val, _ := d.Int(arg.LName())
		value = reflect.ValueOf(val)
	case DATATYPE_INT64:
		val, _ := d.Int64(arg.LName())
		value = reflect.ValueOf(val)
	case DATATYPE_FLOAT:
		val, _ := d.Float32(arg.LName())
		value = reflect.ValueOf(val)
	case DATATYPE_FLOAT64:
		val, _ := d.Float64(arg.LName())
		value = reflect.ValueOf(val)
	case DATATYPE_EMAIL:
		val, _ := d.Email(arg.LName())
		value = reflect.ValueOf(val)
	case DATATYPE_URL:
		val, _ := d.Url(arg.LName())
		value = reflect.ValueOf(val)
	case DATATYPE_DATE:
		val, _ := d.Date(arg.LName())
		value = reflect.ValueOf(val)
	case DATATYPE_TIME:
		val, _ := d.Time(arg.LName())
		value = reflect.ValueOf(val)
	case DATATYPE_DATETIME:
		val, _ := d.DateTime(arg.LName())
		value = reflect.ValueOf(val)
	case DATATYPE_STRUCT:
		fallthrough
	case DATATYPE_EMBEDDED:
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
