package view

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/web"
)

type JsonView mvc.VM

func (jv JsonView) ContentType(ctxt *web.Context) string {
	return "application/json; charset=utf-8"
}

func (jv JsonView) Render(ctxt *web.Context) ([]byte, error) {
	if jv == nil || len(jv) == 0 {
		return []byte{'{', '}'}, nil
	}

	buf := bytes.NewBuffer(make([]byte, 0, 100))
	jsone := json.NewEncoder(buf)
	if err := jsone.Encode(jv.getEncodingInterface()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (jv JsonView) RenderString(ctxt *web.Context) (string, error) {
	b, err := jv.Render(ctxt)
	if err == nil {
		return string(b), nil
	}
	return "", err
}

func (jv JsonView) Publish(ctxt *web.Context) error {
	ctxt.SetHeader("Content-Type", "application/json; charset=utf-8")

	if jv == nil || len(jv) == 0 {
		ctxt.Response.Write([]byte{'{', '}'})
		return nil
	}

	jsone := json.NewEncoder(ctxt.Response)
	if err := jsone.Encode(jv.getEncodingInterface()); err != nil {
		return err
	}
	return nil
}

func (jv JsonView) getEncodingInterface() (i interface{}) {
	i = jv

	if len(jv) == 1 && jv[""] != nil {
		t := reflect.TypeOf(jv[""])

		if kind := t.Kind(); kind == reflect.Map || kind == reflect.Struct ||
			(kind == reflect.Ptr && t.Elem().Kind() == reflect.Struct) {
			i = jv[""]
		}
	}

	return
}

type FullJsonView JsonView

func (jv FullJsonView) ContentType(ctxt *web.Context) string {
	return JsonView(jv).ContentType(ctxt)
}

func (jv FullJsonView) Render(ctxt *web.Context) ([]byte, error) {
	return JsonView(jv).Render(ctxt)
}

func (jv FullJsonView) RenderString(ctxt *web.Context) (string, error) {
	return JsonView(jv).RenderString(ctxt)
}

func (jv FullJsonView) Publish(ctxt *web.Context) error {
	return JsonView(jv).Publish(ctxt)
}

func (jv *FullJsonView) SetVMap(vmap ...mvc.VM) {
	if *jv == nil {
		*jv = NewFullJsonView(nil)
	}

	_vmap := *jv

	for _, vm := range vmap {
		for k, v := range vm {
			_vmap[k] = v
		}
	}
}

func (jv *FullJsonView) SetData(name string, value interface{}) {
	if *jv == nil {
		*jv = NewFullJsonView(nil)
	}

	_vmap := *jv
	_vmap[name] = value
}

func NewJsonView(vmap mvc.VM) JsonView {
	if vmap == nil {
		vmap = mvc.VM{}
	}
	return JsonView(vmap)
}

func NewFullJsonView(vmap mvc.VM) FullJsonView {
	if vmap == nil {
		vmap = mvc.VM{}
	}
	return FullJsonView(NewJsonView(vmap))
}
