package controller

import (
	"net/http"
	"reflect"

	"github.com/zaolab/sunnified/web"
)

type Type int

const (
	_ Type = iota
	ContypeFunc
	ContypeStruct
	ContypeConstructor
	ContypeScontroller
)

const (
	SconConstructName = "Construct_"
	SconDestructName  = "Destruct_"
)

type ReqMethod uint16

const (
	// the first four must be get, post, put and delete (the order is less impt)
	// since ActionMap.Add() depends on it to work correctly
	ReqMethodGet ReqMethod = 1 << iota
	ReqMethodPost
	ReqMethodPut
	ReqMethodDelete
	ReqMethodPatch
	ReqMethodOptions
	ReqMethodHead

	ReqMethodCommon ReqMethod = 15  //1 | 2 | 4 | 8
	ReqMethodAll    ReqMethod = 127 //1 | 2 | 4 | 8 | 16 | 32 | 64
)

func GetReqMethod(r *http.Request) ReqMethod {
	switch r.Method {
	case "GET":
		return ReqMethodGet
	case "POST":
		return ReqMethodPost
	case "PUT":
		return ReqMethodPut
	case "DELETE":
		return ReqMethodDelete
	case "PATCH":
		return ReqMethodPatch
	case "OPTIONS":
		return ReqMethodOptions
	case "HEAD":
		return ReqMethodHead
	}

	return ReqMethod(0)
}

func GetXReqMethod(ctxt *web.Context) ReqMethod {
	var xreq = ctxt.XMethod()

	switch xreq {
	case "POST":
		return ReqMethodPost
	case "PUT":
		return ReqMethodPut
	case "DELETE":
		return ReqMethodDelete
	default:
	}

	return ReqMethodGet
}

type ActionMap map[string]map[ReqMethod]*ActionMeta

type Meta struct {
	name    string
	modname string
	rtype   reflect.Type
	meths   ActionMap
	reqmeth ReqMethod
	args    []*ArgMeta
	fields  []*FieldMeta
	t       Type
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
		for rmeth := range actions {
			rm = rm | rmeth
		}
	}
	return
}

func (a ActionMap) GetReqMethList(name string) (rml []string) {
	if actions, exists := a[name]; exists {
		rml = make([]string, 0, len(actions))

		for rmeth := range actions {
			switch rmeth {
			case ReqMethodGet:
				rml = append(rml, "GET")
			case ReqMethodPost:
				rml = append(rml, "POST")
			case ReqMethodPut:
				rml = append(rml, "PUT")
			case ReqMethodDelete:
				rml = append(rml, "DELETE")
			case ReqMethodPatch:
				rml = append(rml, "PATCH")
			case ReqMethodHead:
				rml = append(rml, "HEAD")
			case ReqMethodOptions:
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

func (cm *Meta) Action(name string, reqtype ReqMethod) *ActionMeta {
	return cm.meths.Get(name, reqtype)
}

func (cm Meta) ActionFromRequest(name string, ctxt *web.Context) *ActionMeta {
	return cm.Action(name, GetXReqMethod(ctxt))
}

func (cm Meta) ActionAvailableMethods(name string) ReqMethod {
	return cm.meths.GetReqMeth(name)
}

func (cm Meta) ActionAvailableMethodsList(name string) []string {
	return cm.meths.GetReqMethList(name)
}

func (cm *Meta) New() reflect.Value {
	return reflect.New(cm.rtype)
}

func (cm *Meta) HasActionMethod(name string, reqtype ReqMethod) bool {
	return cm.Action(name, reqtype) != nil
}

func (cm *Meta) HasAction(name string) bool {
	return cm.meths.HasAction(name)
}

func (cm *Meta) Name() string {
	return cm.name
}

func (cm *Meta) Module() string {
	return cm.modname
}

func (cm *Meta) RType() reflect.Type {
	return cm.rtype
}

func (cm *Meta) Meths() ActionMap {
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

func (cm *Meta) ReqMeth() ReqMethod {
	return cm.reqmeth
}

func (cm *Meta) Args() []*ArgMeta {
	out := make([]*ArgMeta, len(cm.args))
	copy(out, cm.args)
	return out
}

func (cm *Meta) Fields() []*FieldMeta {
	out := make([]*FieldMeta, len(cm.fields))
	copy(out, cm.fields)
	return out
}

func (cm *Meta) T() Type {
	return cm.t
}
