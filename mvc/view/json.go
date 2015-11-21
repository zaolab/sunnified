package view

import (
	"bytes"
	"encoding/json"
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/web"
	"reflect"
)

type JsonView mvc.VM

func (this JsonView) ContentType(ctxt *web.Context) string {
	return "application/json; charset=utf-8"
}

func (this JsonView) Render(ctxt *web.Context) ([]byte, error) {
	if this == nil || len(this) == 0 {
		return []byte{'{', '}'}, nil
	}

	buf := bytes.NewBuffer(make([]byte, 0, 100))
	jsone := json.NewEncoder(buf)
	if err := jsone.Encode(this.getEncodingInterface()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (this JsonView) RenderString(ctxt *web.Context) (string, error) {
	b, err := this.Render(ctxt)
	if err == nil {
		return string(b), nil
	}
	return "", err
}

func (this JsonView) Publish(ctxt *web.Context) error {
	ctxt.SetHeader("Content-Type", "application/json; charset=utf-8")

	if this == nil || len(this) == 0 {
		ctxt.Response.Write([]byte{'{', '}'})
		return nil
	}

	jsone := json.NewEncoder(ctxt.Response)
	if err := jsone.Encode(this.getEncodingInterface()); err != nil {
		return err
	}
	return nil
}

func (this JsonView) getEncodingInterface() (i interface{}) {
	i = this

	if len(this) == 1 && this[""] != nil {
		t := reflect.TypeOf(this[""])

		if kind := t.Kind(); kind == reflect.Map || kind == reflect.Struct ||
			(kind == reflect.Ptr && t.Elem().Kind() == reflect.Struct) {
			i = this[""]
		}
	}

	return
}

type FullJsonView JsonView

func (this FullJsonView) ContentType(ctxt *web.Context) string {
	return JsonView(this).ContentType(ctxt)
}

func (this FullJsonView) Render(ctxt *web.Context) ([]byte, error) {
	return JsonView(this).Render(ctxt)
}

func (this FullJsonView) RenderString(ctxt *web.Context) (string, error) {
	return JsonView(this).RenderString(ctxt)
}

func (this FullJsonView) Publish(ctxt *web.Context) error {
	return JsonView(this).Publish(ctxt)
}

func (this *FullJsonView) SetVMap(vmap ...mvc.VM) {
	if *this == nil {
		*this = NewFullJsonView(nil)
	}

	_vmap := *this

	for _, vm := range vmap {
		for k, v := range vm {
			_vmap[k] = v
		}
	}
}

func (this *FullJsonView) SetData(name string, value interface{}) {
	if *this == nil {
		*this = NewFullJsonView(nil)
	}

	_vmap := *this
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
