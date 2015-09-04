package view

import (
	"bytes"
	"compress/gzip"
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/web"
	"html/template"
	"io"
	"log"
	"net/http"
)

type MultiView struct {
	mvc.VM
	GetTmpl func(fmap template.FuncMap) (t *template.Template, err error)
	fmap    template.FuncMap
}

func (this *MultiView) SetViewFunc(name string, f interface{}) {
	if this.fmap == nil {
		this.fmap = template.FuncMap{}
	}
	this.fmap[name] = f
}

func (this *MultiView) SetViewFuncName(name string) {
	if this.fmap == nil {
		this.fmap = template.FuncMap{}
	}

	if _, exists := this.fmap[name]; !exists {
		this.fmap[name] = func(i ...interface{}) string { return "" }
	}
}

func (this *MultiView) SetGetTmpl(f func(fmap template.FuncMap) (t *template.Template, err error)) {
	this.GetTmpl = f
}

func (this *MultiView) SetVMap(vmap ...mvc.VM) {
	if this.VM == nil {
		this.VM = mvc.VM{}
	}
	for _, vm := range vmap {
		for k, v := range vm {
			this.VM[k] = v
		}
	}
}

func (this *MultiView) SetData(name string, value interface{}) {
	if this.VM == nil {
		this.VM = mvc.VM{}
	}
	this.VM[name] = value
}

func (this *MultiView) ContentType(ctxt *web.Context) string {
	return "text/html; charset=utf-8"
}

func (this *MultiView) getTmpl(names mvc.MvcMeta) (tmpl *template.Template, ext string, err error) {
	if this.GetTmpl != nil {
		tmpl, err = this.GetTmpl(this.fmap)
	} else {
		tmpl, err = mvc.GetHtmlTmpl(mvc.GetTemplateRelPath(names, ext), this.fmap)
	}

	return
}

func (this *MultiView) Render(ctxt *web.Context) (b []byte, err error) {
	if tmpl, _, err := this.getTmpl(mvc.GetMvcMeta(ctxt)); err == nil {
		buf := &bytes.Buffer{}
		tmpl.Execute(buf, this.VM)
		b = buf.Bytes()
	}

	return
}

func (this *MultiView) RenderString(ctxt *web.Context) (s string, err error) {
	var b []byte
	b, err = this.Render(ctxt)
	if err == nil {
		s = string(b)
	}
	return
}

func (this *MultiView) Publish(ctxt *web.Context) (err error) {
	names := mvc.GetMvcMeta(ctxt)
	if names[mvc.MVC_ACTION] == "" {
		names[mvc.MVC_ACTION] = "_"
	}

	var tmpl *template.Template
	tmpl, _, err = this.getTmpl(names)

	if err == nil {
		var method = ctxt.Method()

		ctxt.SetHeader("Content-Type", "text/html; charset=utf-8")

		if method != "HEAD" {
			var err error
			var tw io.Writer = ctxt.Response
			var gzipwriter *gzip.Writer

			if ctxt.ReqHeaderHas("Accept-Encoding", "gzip") {
				ctxt.SetHeader("Content-Encoding", "gzip")
				gzipwriter, _ = gzip.NewWriterLevel(ctxt.Response, gzip.BestSpeed)
				tw = gzipwriter
			}

			ctxt.SetHeader("Vary", "Accept-Encoding")
			ctxt.Response.WriteHeader(200)

			err = tmpl.Execute(tw, this.VM)

			if err != nil {
				// Header already sent... multiple write headers
				//panic(err)
				log.Println(err)
			}

			if gzipwriter != nil {
				gzipwriter.Close()
			}

			if flushw, ok := ctxt.Response.(http.Flusher); ok {
				flushw.Flush()
			}
		} else {
			ctxt.Response.WriteHeader(200)
		}
	} else {
		log.Println(err)
		ctxt.SetErrorCode(500)
	}

	return
}
