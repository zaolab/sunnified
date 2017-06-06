package view

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/web"
)

var (
	emptyJSON    = []byte{'{', '}'}
	emptyJSONLen = strconv.Itoa(len(emptyJSON))
)

type JSONView mvc.VM

func (jv JSONView) ContentType(ctxt *web.Context) string {
	return "application/json; charset=utf-8"
}

func (jv JSONView) Render(ctxt *web.Context) ([]byte, error) {
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

func (jv JSONView) RenderString(ctxt *web.Context) (string, error) {
	b, err := jv.Render(ctxt)
	if err == nil {
		return string(b), nil
	}
	return "", err
}

func (jv JSONView) Publish(ctxt *web.Context) error {
	ctxt.SetHeader("Content-Type", "application/json; charset=utf-8")

	var meth = ctxt.Method()

	if meth == "HEAD" {
		if jv == nil || len(jv) == 0 {
			ctxt.SetHeader("Content-Length", emptyJSONLen)
		}

		ctxt.Response.WriteHeader(200)
		return nil
	}

	if jv == nil || len(jv) == 0 {
		ctxt.SetHeader("Content-Length", emptyJSONLen)
		ctxt.Response.Write(emptyJSON)
		return nil
	}

	var (
		bb, err = json.Marshal(jv.getEncodingInterface())
		etag    string
	)

	if err != nil {
		return err
	}

	if meth == "GET" && ctxt.ResHeader("Cache-Control") == "" && ctxt.ResHeader("ETag") == "" {
		ctxt.SetHeader("Cache-Control", "max-age=0, must-revalidate")
		etag = fmt.Sprintf("%x", md5.Sum(bb))
	}

	if msize := GetMinGZIPSize(); msize <= 0 || len(bb) < msize {
		goto skipgzip
	}

	ctxt.AddHeaderVary("Accept-Encoding")

	if ctxt.ReqHeaderHas("Accept-Encoding", "gzip") {
		if etag != "" {
			if ctxt.IfNoneMatch(`W/"`+etag+`.gzip"`) == 304 {
				ctxt.Response.WriteHeader(304)
				return nil
			}
		}

		gw, err := gzip.NewWriterLevel(ctxt.Response, gzip.BestSpeed)
		if err != nil {
			goto skipgzip
		}

		ctxt.SetHeader("Content-Encoding", "gzip")

		if _, err = gw.Write(bb); err != nil {
			return err
		}

		return gw.Close()
	}

skipgzip:
	if etag != "" && ctxt.IfNoneMatch(`"`+etag+`"`) == 304 {
		ctxt.Response.WriteHeader(304)
		return nil
	}

	ctxt.SetHeader("Content-Length", strconv.Itoa(len(bb)))

	_, err = ctxt.Response.Write(bb)
	return err
}

func (jv JSONView) getEncodingInterface() (i interface{}) {
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

type FullJSONView JSONView

func (jv FullJSONView) ContentType(ctxt *web.Context) string {
	return JSONView(jv).ContentType(ctxt)
}

func (jv FullJSONView) Render(ctxt *web.Context) ([]byte, error) {
	return JSONView(jv).Render(ctxt)
}

func (jv FullJSONView) RenderString(ctxt *web.Context) (string, error) {
	return JSONView(jv).RenderString(ctxt)
}

func (jv FullJSONView) Publish(ctxt *web.Context) error {
	return JSONView(jv).Publish(ctxt)
}

func (jv *FullJSONView) SetVMap(vmap ...mvc.VM) {
	if *jv == nil {
		*jv = NewFullJSONView(nil)
	}

	_vmap := *jv

	for _, vm := range vmap {
		for k, v := range vm {
			_vmap[k] = v
		}
	}
}

func (jv *FullJSONView) SetData(name string, value interface{}) {
	if *jv == nil {
		*jv = NewFullJSONView(nil)
	}

	_vmap := *jv
	_vmap[name] = value
}

func NewJSONView(vmap mvc.VM) JSONView {
	if vmap == nil {
		vmap = mvc.VM{}
	}
	return JSONView(vmap)
}

func NewFullJSONView(vmap mvc.VM) FullJSONView {
	if vmap == nil {
		vmap = mvc.VM{}
	}
	return FullJSONView(NewJSONView(vmap))
}
