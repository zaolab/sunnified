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
	type_responsewriter     = reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()
	type_request            = reflect.TypeOf((*http.Request)(nil))
	type_mvc_view           = reflect.TypeOf((*mvc.View)(nil)).Elem()
	type_mvc_controller     = reflect.TypeOf((*mvc.Controller)(nil)).Elem()
	type_slicestring        = reflect.TypeOf([]string{})
	type_mapstringstring    = reflect.TypeOf(map[string]string{})
	type_mapstringinterface = reflect.TypeOf(map[string]interface{}{})
	type_vmap               = reflect.TypeOf(mvc.VM{})
	type_upath              = reflect.TypeOf(web.UPath{})
	type_pdata              = reflect.TypeOf(web.PData{})
	type_statuscode         = reflect.TypeOf((web.StatusCode)(0))
	type_timetime           = reflect.TypeOf((*time.Time)(nil))
	type_timeduration       = reflect.TypeOf((*time.Duration)(nil))
	type_webcontext         = reflect.TypeOf((*web.Context)(nil))

	len_argtype_string_suffix   = len(DATATYPE_STRING_SUFFIX)
	len_argtype_bool_suffix     = len(DATATYPE_BOOL_SUFFIX)
	len_argtype_int_suffix      = len(DATATYPE_INT_SUFFIX)
	len_argtype_int64_suffix    = len(DATATYPE_INT64_SUFFIX)
	len_argtype_float_suffix    = len(DATATYPE_FLOAT_SUFFIX)
	len_argtype_float64_suffix  = len(DATATYPE_FLOAT64_SUFFIX)
	len_argtype_email_suffix    = len(DATATYPE_EMAIL_SUFFIX)
	len_argtype_url_suffix      = len(DATATYPE_URL_SUFFIX)
	len_argtype_date_suffix     = len(DATATYPE_DATE_SUFFIX)
	len_argtype_time_suffix     = len(DATATYPE_TIME_SUFFIX)
	len_argtype_datetime_suffix = len(DATATYPE_DATETIME_SUFFIX)

	len_form_valuetype_lprefix = len(FORM_VALUETYPE_LPREFIX)

	group = NewControllerGroup()
)

func NewControllerGroup() *ControllerGroup {
	return &ControllerGroup{
		details:  make(map[string]map[string]*ControllerMeta),
		modules:  make(map[string]string),
		detmutex: sync.RWMutex{},
		modmutex: sync.RWMutex{},
	}
}

func GetDefaultControllerGroup() *ControllerGroup {
	return group
}

type ControllerGroup struct {
	details  map[string]map[string]*ControllerMeta
	modules  map[string]string
	detmutex sync.RWMutex
	modmutex sync.RWMutex
}

func (cg *ControllerGroup) HasModule(mod string) bool {
	cg.detmutex.RLock()
	defer cg.detmutex.RUnlock()

	_, exists := cg.details[mod]
	return exists
}

func (cg *ControllerGroup) Module(mod string) (m map[string]*ControllerMeta) {
	cg.detmutex.RLock()
	defer cg.detmutex.RUnlock()

	mod = strings.ToLower(mod)
	if md, ok := cg.details[mod]; ok {
		m = md
	}

	return
}

func (cg *ControllerGroup) HasController(mod, con string) (exists bool) {
	cg.detmutex.RLock()
	defer cg.detmutex.RUnlock()

	if _, exists = cg.details[mod]; exists {
		con, _ = parseReqMethod(con)
		_, exists = cg.details[mod][con]
	}

	return
}

func (cg *ControllerGroup) Controller(mod, con string) (c *ControllerMeta) {
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

func (cg *ControllerGroup) AddModule(alias string, modname string) {
	cg.modmutex.Lock()
	defer cg.modmutex.Unlock()
	cg.modules[alias] = modname
}

func (cg *ControllerGroup) AddController(cinterface interface{}) (string, string) {
	cg.detmutex.Lock()
	defer cg.detmutex.Unlock()

	cm, controller, alias, modname := MakeControllerMeta(cinterface)

	if cg.details[alias] == nil {
		cg.details[alias] = make(map[string]*ControllerMeta)
	} else if c, exists := cg.details[alias][controller]; exists {
		if cm.rtype == c.rtype {
			return alias, controller
		} else {
			panic("Duplicate controller name: " + alias + "." + controller)
		}
	}

	cg.createModAlias(alias, modname)
	cg.details[alias][controller] = cm
	return alias, controller
}

func (cg *ControllerGroup) createModAlias(alias, modname string) {
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

func Module(mod string) (m map[string]*ControllerMeta) {
	return group.Module(mod)
}

func HasController(mod, con string) (exists bool) {
	return group.HasController(mod, con)
}

func Controller(mod, con string) *ControllerMeta {
	return group.Controller(mod, con)
}

func AddModule(alias string, modname string) {
	group.AddModule(alias, modname)
}

func AddController(cinterface interface{}) (string, string) {
	return group.AddController(cinterface)
}

func MakeControllerMeta(cinterface interface{}) (cm *ControllerMeta, ctrlname, mod, modfull string) {
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

	cm = &ControllerMeta{
		name:    ownname,
		modname: mod,
		rtype:   rawtype,
		meths:   make(ActionMap),
		reqmeth: reqmeth,
	}

	switch rtype.Kind() {
	case reflect.Func:
		cm.t = CONTYPE_FUNC
		cm.args = parseArgsMeta(rtype, false)

		var isconstruct bool
		if cm.ResultStyle, isconstruct = parseResultStyle(rtype, true); isconstruct {
			cm.t = CONTYPE_CONSTRUCTOR
			cm.name = rtype.Name()
			// TODO: the package might be different,
			// but we assumes it falls in the same package (ignore it for now)
		}
	case reflect.Struct:
		if rawtype.Implements(type_mvc_controller) {
			cm.t = CONTYPE_SCONTROLLER
		} else {
			cm.t = CONTYPE_STRUCT
		}

		cm.fields = parseFieldsMeta(rtype)
	}

	for i, count := 0, rawtype.NumMethod(); i < count; i++ {
		meth := rawtype.Method(i)

		_, ismvcmethod := type_mvc_controller.MethodByName(meth.Name)
		if ismvcmethod || meth.Name[len(meth.Name)-1] == '_' {
			continue
		}
		_, ismvcmethod = type_webcontext.MethodByName(meth.Name)
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
			if out.Implements(type_mvc_view) {
				rs.view = true
				// if the function outputs a struct,
				// we will consider it a constructor function,
				// and the real controller is its output
			} else if iscontrol && numout == 1 {
				rtype = out
				isconstruct = true
				rs, _ = parseResultStyle(rtype, false)
			}
		} else if out == type_mvc_view {
			rs.view = true
		} else if numout == 1 && out == type_statuscode {
			rs.status = true
		} else if out == type_vmap {
			rs.vmap = true
		} else if out == type_mapstringinterface {
			rs.mapsi = true
		} else if out.Implements(type_mvc_view) {
			rs.view = true
		}

		// check for status output
		if numout == 2 {
			out := rtype.Out(1)

			if out.Kind() == reflect.Int && out == type_statuscode {
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
		case arg == type_webcontext:
			argmeta.rtype = type_webcontext
			argmeta.t = DATATYPE_WEBCONTEXT
		case arg == type_responsewriter:
			argmeta.rtype = type_responsewriter
			argmeta.t = DATATYPE_RESPONSEWRITER
		case arg == type_request:
			argmeta.rtype = type_request
			argmeta.t = DATATYPE_REQUEST
		case arg == type_upath:
			argmeta.t = DATATYPE_UPATH
		case arg == type_slicestring:
			argmeta.t = DATATYPE_UPATH_SLICE
		case arg == type_pdata:
			argmeta.t = DATATYPE_PDATA
		case arg == type_mapstringstring:
			argmeta.t = DATATYPE_PDATA_MAP
		case argkind == reflect.Struct:
			switch {
			case strings.HasSuffix(argname, DATATYPE_DATE_SUFFIX):
				argmeta.t = DATATYPE_DATE
				argmeta.lname = argmeta.lname[:namelen-len_argtype_date_suffix]
			case strings.HasSuffix(argname, DATATYPE_TIME_SUFFIX):
				argmeta.t = DATATYPE_TIME
				argmeta.lname = argmeta.lname[:namelen-len_argtype_time_suffix]
			case strings.HasSuffix(argname, DATATYPE_DATETIME_SUFFIX):
				argmeta.t = DATATYPE_DATETIME
				argmeta.lname = argmeta.lname[:namelen-len_argtype_datetime_suffix]
			default:
				argmeta.t = DATATYPE_STRUCT
				argmeta.fields = parseFieldsMeta(arg)
			}
		case argkind == reflect.String:
			switch {
			case strings.HasSuffix(argname, DATATYPE_STRING_SUFFIX):
				argmeta.t = DATATYPE_STRING
				argmeta.lname = argmeta.lname[:namelen-len_argtype_string_suffix]
			case strings.HasSuffix(argname, DATATYPE_EMAIL_SUFFIX):
				argmeta.t = DATATYPE_EMAIL
				argmeta.lname = argmeta.lname[:namelen-len_argtype_email_suffix]
			case strings.HasSuffix(argname, DATATYPE_URL_SUFFIX):
				argmeta.t = DATATYPE_URL
				argmeta.lname = argmeta.lname[:namelen-len_argtype_url_suffix]
			}
		case argkind == reflect.Int && strings.HasSuffix(argname, DATATYPE_INT_SUFFIX):
			argmeta.t = DATATYPE_INT
			argmeta.lname = argmeta.lname[:namelen-len_argtype_int_suffix]
		case argkind == reflect.Int64 && strings.HasSuffix(argname, DATATYPE_INT64_SUFFIX):
			argmeta.t = DATATYPE_INT64
			argmeta.lname = argmeta.lname[:namelen-len_argtype_int64_suffix]
		case argkind == reflect.Float32 && strings.HasSuffix(argname, DATATYPE_FLOAT_SUFFIX):
			argmeta.t = DATATYPE_FLOAT
			argmeta.lname = argmeta.lname[:namelen-len_argtype_float_suffix]
		case argkind == reflect.Float64 && strings.HasSuffix(argname, DATATYPE_FLOAT64_SUFFIX):
			argmeta.t = DATATYPE_FLOAT64
			argmeta.lname = argmeta.lname[:namelen-len_argtype_float64_suffix]
		case argkind == reflect.Float64 && strings.HasSuffix(argname, DATATYPE_BOOL_SUFFIX):
			argmeta.t = DATATYPE_BOOL
			argmeta.lname = argmeta.lname[:namelen-len_argtype_bool_suffix]
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
		is_form := strings.HasPrefix(lname, FORM_VALUETYPE_LPREFIX)
		is_accepted := field.Type == type_webcontext || field.Type == type_responsewriter || field.Type == type_request || field.Anonymous || field.Tag.Get(FORM_VALUETYPE_TAG_NAME) != ""
		is_parser := field.Tag.Get(STRUCTVALUEFEED_TAG) != "" || field.Tag.Get(STRUCTVALUERES_TAG) != ""

		// only exported fields (where PkgPath == "") can have their values set
		// and fields which have a prefix of form_ or has a type tag
		if field.PkgPath != "" || (!is_accepted && !is_form && !is_parser) {
			continue
		} else if is_form {
			lname = lname[len_form_valuetype_lprefix:]
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

		if is_form || is_accepted {
			switch fkind {
			case reflect.Struct:
				switch field.Type {
				case type_webcontext:
					fmeta.t = DATATYPE_WEBCONTEXT
				case type_responsewriter:
					fmeta.t = DATATYPE_RESPONSEWRITER
				case type_request:
					fmeta.t = DATATYPE_REQUEST
				case type_timetime:
					fmeta.rtype = type_timetime
					if field.Tag.Get(FORM_VALUETYPE_TAG_NAME) == "date" {
						fmeta.t = DATATYPE_DATE
					} else {
						fmeta.t = DATATYPE_DATETIME
					}
				case type_timeduration:
					fmeta.rtype = type_timeduration
					fmeta.t = DATATYPE_TIME
				default:
					if field.Anonymous {
						fmeta.t = DATATYPE_EMBEDDED
					} else {
						fmeta.t = DATATYPE_STRUCT
					}
					fmeta.fields = parseFieldsMeta(field.Type)
				}
			case reflect.String:
				switch field.Tag.Get(FORM_VALUETYPE_TAG_NAME) {
				case "email":
					fmeta.t = DATATYPE_EMAIL
				case "url":
					fmeta.t = DATATYPE_URL
				default:
					fmeta.t = DATATYPE_STRING
				}
			case reflect.Int:
				fmeta.t = DATATYPE_INT
			case reflect.Int64:
				fmeta.t = DATATYPE_INT64
			case reflect.Float32:
				fmeta.t = DATATYPE_FLOAT
			case reflect.Float64:
				fmeta.t = DATATYPE_FLOAT64
			case reflect.Bool:
				fmeta.t = DATATYPE_BOOL
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
	reqmeth = REQMETHOD_COMMON

	switch {
	case strings.HasPrefix(name, "GET"):
		reqmeth = REQMETHOD_GET
		alias = alias[3:]
	case strings.HasPrefix(name, "POST"):
		reqmeth = REQMETHOD_POST
		alias = alias[4:]
	case strings.HasPrefix(name, "PUT"):
		reqmeth = REQMETHOD_PUT
		alias = alias[3:]
	case strings.HasPrefix(name, "DELETE"):
		reqmeth = REQMETHOD_DELETE
		alias = alias[6:]
	}

	if alias == "" {
		alias = "_"
	}

	return
}
