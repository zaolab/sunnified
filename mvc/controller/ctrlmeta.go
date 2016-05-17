package controller

import (
	"net/http"
	"reflect"

	"github.com/zaolab/sunnified/web"
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

func GetXReqMethod(ctxt *web.Context) ReqMethod {
	var xreq = ctxt.XMethod()

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

func (a ActionMap) Add(name string, am *ActionMeta) {
	if _, exists := a[name]; !exists {
		a[name] = make(map[ReqMethod]*ActionMeta)
	}
	for i := uint16(0); i < 4; i++ {
		reqtype := ReqMethod(1 << i)
		if (am.reqmeth & reqtype) == reqtype {
			a[name][reqtype] = am
		}
	}
}

func (a ActionMap) HasAction(name string) bool {
	_, exists := a[name]
	return exists
}

func (a ActionMap) Get(name string, reqtype ReqMethod) *ActionMeta {
	if actions, exists := a[name]; exists {
		if action, exists := actions[reqtype]; exists {
			return action
		}
	}
	return nil
}

func (a ActionMap) GetReqMeth(name string) (rm ReqMethod) {
	if actions, exists := a[name]; exists {
		for rmeth, _ := range actions {
			rm = rm | rmeth
		}
	}
	return
}

func (a ActionMap) GetReqMethList(name string) (rml []string) {
	if actions, exists := a[name]; exists {
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

func (a ActionMap) Delete(name string) {
	delete(a, name)
}

func (a ActionMap) Remove(name string, reqtype ReqMethod) {
	if actions, exists := a[name]; exists {
		delete(actions, reqtype)
	}
}

func (a ActionMap) Count() int {
	return len(a)
}

func (cm *ControllerMeta) Action(name string, reqtype ReqMethod) *ActionMeta {
	return cm.meths.Get(name, reqtype)
}

func (cm ControllerMeta) ActionFromRequest(name string, ctxt *web.Context) *ActionMeta {
	return cm.Action(name, GetXReqMethod(ctxt))
}

func (cm ControllerMeta) ActionAvailableMethods(name string) ReqMethod {
	return cm.meths.GetReqMeth(name)
}

func (cm ControllerMeta) ActionAvailableMethodsList(name string) []string {
	return cm.meths.GetReqMethList(name)
}

func (cm *ControllerMeta) New() reflect.Value {
	return reflect.New(cm.rtype)
}

func (cm *ControllerMeta) HasActionMethod(name string, reqtype ReqMethod) bool {
	return cm.Action(name, reqtype) != nil
}

func (cm *ControllerMeta) HasAction(name string) bool {
	return cm.meths.HasAction(name)
}

func (cm *ControllerMeta) Name() string {
	return cm.name
}

func (cm *ControllerMeta) Module() string {
	return cm.modname
}

func (cm *ControllerMeta) RType() reflect.Type {
	return cm.rtype
}

func (cm *ControllerMeta) Meths() ActionMap {
	meths := make(ActionMap)
	for k, v := range cm.meths {
		reqmeths := make(map[ReqMethod]*ActionMeta)
		for k2, v2 := range v {
			reqmeths[k2] = v2
		}
		meths[k] = reqmeths
	}
	return meths
}

func (cm *ControllerMeta) ReqMeth() ReqMethod {
	return cm.reqmeth
}

func (cm *ControllerMeta) Args() []*ArgMeta {
	out := make([]*ArgMeta, len(cm.args))
	copy(out, cm.args)
	return out
}

func (cm *ControllerMeta) Fields() []*FieldMeta {
	out := make([]*FieldMeta, len(cm.fields))
	copy(out, cm.fields)
	return out
}

func (cm *ControllerMeta) T() ControllerType {
	return cm.t
}
