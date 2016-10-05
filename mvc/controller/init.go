package controller

import (
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/web"
)

var (
	typeResponseWriter     = reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()
	typeRequest            = reflect.TypeOf((*http.Request)(nil))
	typeMVCView            = reflect.TypeOf((*mvc.View)(nil)).Elem()
	typeMVCController      = reflect.TypeOf((*mvc.Controller)(nil)).Elem()
	typeSliceString        = reflect.TypeOf([]string{})
	typeMapStringString    = reflect.TypeOf(map[string]string{})
	typeMapStringInterface = reflect.TypeOf(map[string]interface{}{})
	typeVmap               = reflect.TypeOf(mvc.VM{})
	typeUpath              = reflect.TypeOf(web.UPath{})
	typePdata              = reflect.TypeOf(web.PData{})
	typeStatusCode         = reflect.TypeOf((web.StatusCode)(0))
	typeTimeTime           = reflect.TypeOf((*time.Time)(nil))
	typeTimeDuration       = reflect.TypeOf((*time.Duration)(nil))
	typeWebContext         = reflect.TypeOf((*web.Context)(nil))

	lenArgtypeStringSuffix   = len(DatatypeStringSuffix)
	lenArgtypeBoolSuffix     = len(DatatypeBoolSuffix)
	lenArgtypeIntSuffix      = len(DatatypeIntSuffix)
	lenArgtypeInt64Suffix    = len(DatatypeInt64Suffix)
	lenArgtypeFloatSuffix    = len(DatatypeFloatSuffix)
	lenArgtypeFloat64Suffix  = len(DatatypeFloat64Suffix)
	lenArgtypeEmailSuffix    = len(DatatypeEmailSuffix)
	lenArgtypeURLSuffix      = len(DatatypeURLSuffix)
	lenArgtypeDateSuffix     = len(DatatypeDateSuffix)
	lenArgtypeTimeSuffix     = len(DatatypeTimeSuffix)
	lenArgtypeDateTimeSuffix = len(DatatypeDateTimeSuffix)

	lenFormValueTypeLprefix = len(FormValueTypeLprefix)

	group = NewControllerGroup()
)

func NewControllerGroup() *Group {
	return &Group{
		details:  make(map[string]map[string]*Meta),
		modules:  make(map[string]string),
		detmutex: sync.RWMutex{},
		modmutex: sync.RWMutex{},
	}
}

func GetDefaultControllerGroup() *Group {
	return group
}

type Group struct {
	details  map[string]map[string]*Meta
	modules  map[string]string
	detmutex sync.RWMutex
	modmutex sync.RWMutex
}

func (cg *Group) HasModule(mod string) bool {
	cg.detmutex.RLock()
	defer cg.detmutex.RUnlock()

	_, exists := cg.details[mod]
	return exists
}

func (cg *Group) Module(mod string) (m map[string]*Meta) {
	cg.detmutex.RLock()
	defer cg.detmutex.RUnlock()

	mod = strings.ToLower(mod)
	if md, ok := cg.details[mod]; ok {
		m = md
	}

	return
}

func (cg *Group) HasController(mod, con string) (exists bool) {
	cg.detmutex.RLock()
	defer cg.detmutex.RUnlock()

	if _, exists = cg.details[mod]; exists {
		con, _ = parseReqMethod(con)
		_, exists = cg.details[mod][con]
	}

	return
}

func (cg *Group) Controller(mod, con string) (c *Meta) {
	cg.detmutex.RLock()
	defer cg.detmutex.RUnlock()

	mod = strings.ToLower(mod)
	if _, ok := cg.details[mod]; ok {
		con, _ = parseReqMethod(con)
		if cm, ok := cg.details[mod][con]; ok {
			c = cm
		}
	}

	return
}

func (cg *Group) AddModule(alias string, modname string) {
	cg.modmutex.Lock()
	defer cg.modmutex.Unlock()
	cg.modules[alias] = modname
}

func (cg *Group) AddController(cinterface interface{}) (string, string) {
	cg.detmutex.Lock()
	defer cg.detmutex.Unlock()

	cm, controller, alias, modname := MakeControllerMeta(cinterface)

	if cg.details[alias] == nil {
		cg.details[alias] = make(map[string]*Meta)
	} else if c, exists := cg.details[alias][controller]; exists {
		if cm.rtype == c.rtype {
			return alias, controller
		}

		panic("Duplicate controller name: " + alias + "." + controller)
	}

	cg.createModAlias(alias, modname)
	cg.details[alias][controller] = cm
	return alias, controller
}

func (cg *Group) createModAlias(alias, modname string) {
	cg.modmutex.Lock()
	defer cg.modmutex.Unlock()

	// check to see if module alias extracted already exists
	// if not add it into the global modules var
	if _, ok := cg.modules[alias]; ok && cg.modules[alias] != modname {
		panic("Duplicate module name: " + alias + ", " + cg.modules[alias])
	} else if !ok {
		cg.modules[alias] = modname
	}
}

func HasModule(mod string) bool {
	return group.HasModule(mod)
}

func Module(mod string) (m map[string]*Meta) {
	return group.Module(mod)
}

func HasController(mod, con string) (exists bool) {
	return group.HasController(mod, con)
}

func Controller(mod, con string) *Meta {
	return group.Controller(mod, con)
}

func AddModule(alias string, modname string) {
	group.AddModule(alias, modname)
}

func AddController(cinterface interface{}) (string, string) {
	return group.AddController(cinterface)
}

func MakeControllerMeta(cinterface interface{}) (cm *Meta, ctrlname, mod, modfull string) {
	var (
		reqmeth ReqMethod
		ownname string
		rtype   = reflect.TypeOf(cinterface)
		rawtype = rtype
	)

	if rtype.Kind() == reflect.Ptr {
		rtype = rtype.Elem()
	}

	modfull, ownname = rtype.PkgPath(), strings.ToLower(rtype.Name())

	if slashindex := strings.LastIndex(modfull, "/"); slashindex >= 0 {
		mod = modfull[slashindex+1:]
	} else {
		mod = modfull
	}

	mod = strings.ToLower(mod)

	ctrlname, reqmeth = parseReqMethod(ownname)

	cm = &Meta{
		name:    ownname,
		modname: mod,
		rtype:   rawtype,
		meths:   make(ActionMap),
		reqmeth: reqmeth,
	}

	switch rtype.Kind() {
	case reflect.Func:
		cm.t = ContypeFunc
		cm.args = parseArgsMeta(rtype, false)

		var isconstruct bool
		if cm.ResultStyle, isconstruct = parseResultStyle(rtype, true); isconstruct {
			cm.t = ContypeConstructor
			cm.name = rtype.Name()
			// TODO: the package might be different,
			// but we assumes it falls in the same package (ignore it for now)
		}
	case reflect.Struct:
		if rawtype.Implements(typeMVCController) {
			cm.t = ContypeScontroller
		} else {
			cm.t = ContypeStruct
		}

		cm.fields = parseFieldsMeta(rtype)
	}

	for i, count := 0, rawtype.NumMethod(); i < count; i++ {
		meth := rawtype.Method(i)

		_, ismvcmethod := typeMVCController.MethodByName(meth.Name)
		if ismvcmethod || meth.Name[len(meth.Name)-1] == '_' {
			continue
		}
		_, ismvcmethod = typeWebContext.MethodByName(meth.Name)
		if ismvcmethod {
			continue
		}

		action, reqmeth := parseReqMethod(meth.Name)

		ameta := &ActionMeta{
			name:    meth.Name,
			rmeth:   meth,
			reqmeth: reqmeth,
		}

		if ameta.ResultStyle, _ = parseResultStyle(meth.Type, false); ameta.ResultStyle.IsNil() {
			continue
		}

		ameta.args = parseArgsMeta(meth.Type, rawtype.Kind() == reflect.Ptr)
		cm.meths.Add(action, ameta)
	}

	return
}

func parseResultStyle(rtype reflect.Type, iscontrol bool) (rs ResultStyle, isconstruct bool) {
	numout := rtype.NumOut()

	if numout >= 1 && numout <= 2 {
		out := rtype.Out(0)
		outkind := out.Kind()
		if outkind == reflect.Ptr {
			outkind = out.Elem().Kind()
		}

		if outkind == reflect.Struct {
			if out.Implements(typeMVCView) {
				rs.view = true
				// if the function outputs a struct,
				// we will consider it a constructor function,
				// and the real controller is its output
			} else if iscontrol && numout == 1 {
				rtype = out
				isconstruct = true
				rs, _ = parseResultStyle(rtype, false)
			}
		} else if out == typeMVCView {
			rs.view = true
		} else if numout == 1 && out == typeStatusCode {
			rs.status = true
		} else if out == typeVmap {
			rs.vmap = true
		} else if out == typeMapStringInterface {
			rs.mapsi = true
		} else if out.Implements(typeMVCView) {
			rs.view = true
		}

		// check for status output
		if numout == 2 {
			out := rtype.Out(1)

			if out.Kind() == reflect.Int && out == typeStatusCode {
				rs.status = true
			}
		}
	}

	return
}

func parseArgsMeta(rtype reflect.Type, isptr bool) (args []*ArgMeta) {
	start := 0
	if isptr {
		start = 1
	}
	numin := rtype.NumIn()
	args = make([]*ArgMeta, numin-start)
	argi := 0

	for i := start; i < numin; i++ {
		arg := rtype.In(i)
		argname, argkind := arg.Name(), arg.Kind()

		namelen := len(argname)
		argmeta := &ArgMeta{
			DataMeta: DataMeta{
				name:  argname,
				lname: strings.ToLower(argname),
				rtype: arg,
			},
		}

		switch {
		case arg == typeWebContext:
			argmeta.rtype = typeWebContext
			argmeta.t = DatatypeWebContext
		case arg == typeResponseWriter:
			argmeta.rtype = typeResponseWriter
			argmeta.t = DatatypeResponseWriter
		case arg == typeRequest:
			argmeta.rtype = typeRequest
			argmeta.t = DatatypeRequest
		case arg == typeUpath:
			argmeta.t = DatatypeUpath
		case arg == typeSliceString:
			argmeta.t = DatatypeUpathSlice
		case arg == typePdata:
			argmeta.t = DatatypePdata
		case arg == typeMapStringString:
			argmeta.t = DatatypePdataMap
		case argkind == reflect.Struct:
			switch {
			case strings.HasSuffix(argname, DatatypeDateSuffix):
				argmeta.t = DatatypeDate
				argmeta.lname = argmeta.lname[:namelen-lenArgtypeDateSuffix]
			case strings.HasSuffix(argname, DatatypeTimeSuffix):
				argmeta.t = DatatypeTime
				argmeta.lname = argmeta.lname[:namelen-lenArgtypeTimeSuffix]
			case strings.HasSuffix(argname, DatatypeDateTimeSuffix):
				argmeta.t = DatatypeDateTime
				argmeta.lname = argmeta.lname[:namelen-lenArgtypeDateTimeSuffix]
			default:
				argmeta.t = DatatypeStruct
				argmeta.fields = parseFieldsMeta(arg)
			}
		case argkind == reflect.String:
			switch {
			case strings.HasSuffix(argname, DatatypeStringSuffix):
				argmeta.t = DatatypeString
				argmeta.lname = argmeta.lname[:namelen-lenArgtypeStringSuffix]
			case strings.HasSuffix(argname, DatatypeEmailSuffix):
				argmeta.t = DatatypeEmail
				argmeta.lname = argmeta.lname[:namelen-lenArgtypeEmailSuffix]
			case strings.HasSuffix(argname, DatatypeURLSuffix):
				argmeta.t = DatatypeURL
				argmeta.lname = argmeta.lname[:namelen-lenArgtypeURLSuffix]
			}
		case argkind == reflect.Int && strings.HasSuffix(argname, DatatypeIntSuffix):
			argmeta.t = DatatypeInt
			argmeta.lname = argmeta.lname[:namelen-lenArgtypeIntSuffix]
		case argkind == reflect.Int64 && strings.HasSuffix(argname, DatatypeInt64Suffix):
			argmeta.t = DatatypeInt64
			argmeta.lname = argmeta.lname[:namelen-lenArgtypeInt64Suffix]
		case argkind == reflect.Float32 && strings.HasSuffix(argname, DatatypeFloatSuffix):
			argmeta.t = DatatypeFloat
			argmeta.lname = argmeta.lname[:namelen-lenArgtypeFloatSuffix]
		case argkind == reflect.Float64 && strings.HasSuffix(argname, DatatypeFloat64Suffix):
			argmeta.t = DatatypeFloat64
			argmeta.lname = argmeta.lname[:namelen-lenArgtypeFloat64Suffix]
		case argkind == reflect.Float64 && strings.HasSuffix(argname, DatatypeBoolSuffix):
			argmeta.t = DatatypeBool
			argmeta.lname = argmeta.lname[:namelen-lenArgtypeBoolSuffix]
		}
		args[argi] = argmeta
		argi++
	}

	return
}

func parseFieldsMeta(rtype reflect.Type) (fields []*FieldMeta) {
	numfields := rtype.NumField()
	fields = make([]*FieldMeta, 0, numfields)

	for i := 0; i < numfields; i++ {
		field := rtype.Field(i)
		fieldtype := field.Type
		fname, fkind := field.Name, fieldtype.Kind()
		if fkind == reflect.Ptr {
			fkind = fieldtype.Elem().Kind()
		}

		// TODO: optimise/refactor this crap
		lname := strings.ToLower(fname)
		isForm := strings.HasPrefix(lname, FormValueTypeLprefix)
		isAccepted := field.Type == typeWebContext || field.Type == typeResponseWriter || field.Type == typeRequest || field.Anonymous || field.Tag.Get(FormValueTypeTagName) != ""
		isParser := field.Tag.Get(StructValueFeedTag) != "" || field.Tag.Get(StructValueResTag) != ""

		// only exported fields (where PkgPath == "") can have their values set
		// and fields which have a prefix of form_ or has a type tag
		if field.PkgPath != "" || (!isAccepted && !isForm && !isParser) {
			continue
		} else if isForm {
			lname = lname[lenFormValueTypeLprefix:]
		}

		fmeta := &FieldMeta{
			DataMeta: DataMeta{
				name:  fname,
				lname: lname,
				rtype: field.Type,
			},
			tag:       field.Tag,
			anonymous: field.Anonymous,
		}

		if isForm || isAccepted {
			switch fkind {
			case reflect.Struct:
				switch field.Type {
				case typeWebContext:
					fmeta.t = DatatypeWebContext
				case typeResponseWriter:
					fmeta.t = DatatypeResponseWriter
				case typeRequest:
					fmeta.t = DatatypeRequest
				case typeTimeTime:
					fmeta.rtype = typeTimeTime
					if field.Tag.Get(FormValueTypeTagName) == "date" {
						fmeta.t = DatatypeDate
					} else {
						fmeta.t = DatatypeDateTime
					}
				case typeTimeDuration:
					fmeta.rtype = typeTimeDuration
					fmeta.t = DatatypeTime
				default:
					if field.Anonymous {
						fmeta.t = DatatypeEmbedded
					} else {
						fmeta.t = DatatypeStruct
					}
					fmeta.fields = parseFieldsMeta(field.Type)
				}
			case reflect.String:
				switch field.Tag.Get(FormValueTypeTagName) {
				case "email":
					fmeta.t = DatatypeEmail
				case "url":
					fmeta.t = DatatypeURL
				default:
					fmeta.t = DatatypeString
				}
			case reflect.Int:
				fmeta.t = DatatypeInt
			case reflect.Int64:
				fmeta.t = DatatypeInt64
			case reflect.Float32:
				fmeta.t = DatatypeFloat
			case reflect.Float64:
				fmeta.t = DatatypeFloat64
			case reflect.Bool:
				fmeta.t = DatatypeBool
			}
		}

		if rex := field.Tag.Get("regexp"); rex != "" {
			fmeta.rex = regexp.MustCompile(rex)
		}

		// since some are not exported, we gotten use append instead of directly assigning to i
		fields = append(fields, fmeta)
	}

	return
}

func parseReqMethod(name string) (alias string, reqmeth ReqMethod) {
	alias = strings.ToLower(name)
	reqmeth = ReqMethodCommon

	switch {
	case strings.HasPrefix(name, "GET"):
		reqmeth = ReqMethodGet
		alias = alias[3:]
	case strings.HasPrefix(name, "POST"):
		reqmeth = ReqMethodPost
		alias = alias[4:]
	case strings.HasPrefix(name, "PUT"):
		reqmeth = ReqMethodPut
		alias = alias[3:]
	case strings.HasPrefix(name, "DELETE"):
		reqmeth = ReqMethodDelete
		alias = alias[6:]
	}

	if alias == "" {
		alias = "_"
	}

	return
}
