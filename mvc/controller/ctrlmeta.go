package controller

import (
	"github.com/zaolab/sunnified/web"
	"net/http"
	"reflect"
)

type ControllerType int

const (
	_ ControllerType = iota
	CONTYPE_FUNC
	CONTYPE_STRUCT
	CONTYPE_CONSTRUCTOR
	CONTYPE_SCONTROLLER
)

const (
	SCON_CONSTRUCT_NAME = "Construct_"
	SCON_DESTRUCT_NAME  = "Destruct_"
)

type ReqMethod uint16

const (
	// the first four must be get, post, put and delete (the order is less impt)
	// since ActionMap.Add() depends on it to work correctly
	REQMETHOD_GET ReqMethod = 1 << iota
	REQMETHOD_POST
	REQMETHOD_PUT
	REQMETHOD_DELETE
	REQMETHOD_PATCH
	REQMETHOD_OPTIONS
	REQMETHOD_HEAD

	REQMETHOD_COMMON ReqMethod = 15  //1 | 2 | 4 | 8
	REQMETHOD_ALL    ReqMethod = 127 //1 | 2 | 4 | 8 | 16 | 32 | 64
)

func GetReqMethod(r *http.Request) ReqMethod {
	switch r.Method {
	case "GET":
		return REQMETHOD_GET
	case "POST":
		return REQMETHOD_POST
	case "PUT":
		return REQMETHOD_PUT
	case "DELETE":
		return REQMETHOD_DELETE
	case "PATCH":
		return REQMETHOD_PATCH
	case "OPTIONS":
		return REQMETHOD_OPTIONS
	case "HEAD":
		return REQMETHOD_HEAD
	}

	return ReqMethod(0)
}

func GetXReqMethod(r *http.Request) ReqMethod {
	var xreq = web.XMethod(r)

	switch xreq {
	case "POST":
		return REQMETHOD_POST
	case "PUT":
		return REQMETHOD_PUT
	case "DELETE":
		return REQMETHOD_DELETE
	default:
	}

	return REQMETHOD_GET
}

type ActionMap map[string]map[ReqMethod]*ActionMeta

type ControllerMeta struct {
	name    string
	modname string
	rtype   reflect.Type
	meths   ActionMap
	reqmeth ReqMethod
	args    []*ArgMeta
	fields  []*FieldMeta
	t       ControllerType
	ResultStyle
}

func (this ActionMap) Add(name string, am *ActionMeta) {
	if _, exists := this[name]; !exists {
		this[name] = make(map[ReqMethod]*ActionMeta)
	}
	for i := uint16(0); i < 4; i++ {
		reqtype := ReqMethod(1 << i)
		if (am.reqmeth & reqtype) == reqtype {
			this[name][reqtype] = am
		}
	}
}

func (this ActionMap) HasAction(name string) bool {
	_, exists := this[name]
	return exists
}

func (this ActionMap) Get(name string, reqtype ReqMethod) *ActionMeta {
	if actions, exists := this[name]; exists {
		if action, exists := actions[reqtype]; exists {
			return action
		}
	}
	return nil
}

func (this ActionMap) GetReqMeth(name string) (rm ReqMethod) {
	if actions, exists := this[name]; exists {
		for rmeth, _ := range actions {
			rm = rm | rmeth
		}
	}
	return
}

func (this ActionMap) GetReqMethList(name string) (rml []string) {
	if actions, exists := this[name]; exists {
		rml = make([]string, 0, len(actions))

		for rmeth, _ := range actions {
			switch rmeth {
			case REQMETHOD_GET:
				rml = append(rml, "GET")
			case REQMETHOD_POST:
				rml = append(rml, "POST")
			case REQMETHOD_PUT:
				rml = append(rml, "PUT")
			case REQMETHOD_DELETE:
				rml = append(rml, "DELETE")
			case REQMETHOD_PATCH:
				rml = append(rml, "PATCH")
			case REQMETHOD_HEAD:
				rml = append(rml, "HEAD")
			case REQMETHOD_OPTIONS:
				rml = append(rml, "OPTIONS")
			}
		}
	} else {
		rml = make([]string, 0)
	}

	return
}

func (this ActionMap) Delete(name string) {
	delete(this, name)
}

func (this ActionMap) Remove(name string, reqtype ReqMethod) {
	if actions, exists := this[name]; exists {
		delete(actions, reqtype)
	}
}

func (this ActionMap) Count() int {
	return len(this)
}

func (this *ControllerMeta) Action(name string, reqtype ReqMethod) *ActionMeta {
	return this.meths.Get(name, reqtype)
}

func (this ControllerMeta) ActionFromRequest(name string, r *http.Request) *ActionMeta {
	return this.Action(name, GetXReqMethod(r))
}

func (this ControllerMeta) ActionAvailableMethods(name string) ReqMethod {
	return this.meths.GetReqMeth(name)
}

func (this ControllerMeta) ActionAvailableMethodsList(name string) []string {
	return this.meths.GetReqMethList(name)
}

func (this *ControllerMeta) New() reflect.Value {
	return reflect.New(this.rtype)
}

func (this *ControllerMeta) HasActionMethod(name string, reqtype ReqMethod) bool {
	return this.Action(name, reqtype) != nil
}

func (this *ControllerMeta) HasAction(name string) bool {
	return this.meths.HasAction(name)
}

func (this *ControllerMeta) Name() string {
	return this.name
}

func (this *ControllerMeta) Module() string {
	return this.modname
}

func (this *ControllerMeta) RType() reflect.Type {
	return this.rtype
}

func (this *ControllerMeta) Meths() ActionMap {
	meths := make(ActionMap)
	for k, v := range this.meths {
		reqmeths := make(map[ReqMethod]*ActionMeta)
		for k2, v2 := range v {
			reqmeths[k2] = v2
		}
		meths[k] = reqmeths
	}
	return meths
}

func (this *ControllerMeta) ReqMeth() ReqMethod {
	return this.reqmeth
}

func (this *ControllerMeta) Args() []*ArgMeta {
	out := make([]*ArgMeta, len(this.args))
	copy(out, this.args)
	return out
}

func (this *ControllerMeta) Fields() []*FieldMeta {
	out := make([]*FieldMeta, len(this.fields))
	copy(out, this.fields)
	return out
}

func (this *ControllerMeta) T() ControllerType {
	return this.t
}
