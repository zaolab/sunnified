package view

import (
	"bytes"
	"encoding/json"
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/web"
)

type JsonView struct {
	mvc.VM
}

func (this *JsonView) ContentType(ctxt *web.Context) string {
	return "application/json; charset=utf-8"
}

func (this *JsonView) Render(ctxt *web.Context) ([]byte, error) {
	if this.VM == nil {
		this.VM = mvc.VM{}
	}
	buf := bytes.NewBuffer(make([]byte, 0, 100))
	jsone := json.NewEncoder(buf)
	if err := jsone.Encode(this.VM); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (this *JsonView) RenderString(ctxt *web.Context) (string, error) {
	b, err := this.Render(ctxt)
	if err == nil {
		return string(b), nil
	}
	return "", err
}

func (this *JsonView) Publish(ctxt *web.Context) error {
	if this.VM == nil {
		this.VM = mvc.VM{}
	}
	ctxt.SetHeader("Content-Type", "application/json; charset=utf-8")
	jsone := json.NewEncoder(ctxt.Response)
	if err := jsone.Encode(this.VM); err != nil {
		return err
	}
	return nil
}

type FullJsonView struct {
	*JsonView
}

func (this *FullJsonView) SetVMap(vmap ...mvc.VM) {
	if this.JsonView == nil {
		this.JsonView = NewJsonView(nil)
	}
	for _, vm := range vmap {
		for k, v := range vm {
			this.VM[k] = v
		}
	}
}

func (this *FullJsonView) SetData(name string, value interface{}) {
	if this.JsonView == nil {
		this.JsonView = NewJsonView(nil)
	}
	this.VM[name] = value
}

func NewJsonView(vmap mvc.VM) *JsonView {
	if vmap == nil {
		vmap = mvc.VM{}
	}
	return &JsonView{vmap}
}

func NewFullJsonView(vmap mvc.VM) *FullJsonView {
	if vmap == nil {
		vmap = mvc.VM{}
	}
	return &FullJsonView{NewJsonView(vmap)}
}
