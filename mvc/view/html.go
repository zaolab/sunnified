package view

import (
	"bytes"
	"compress/gzip"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/web"
)

type HtmlView struct {
	mvc.VM
	GetTmpl func(fmap template.FuncMap) (t *template.Template, err error)
	fmap    template.FuncMap
}

func (hv *HtmlView) SetViewFunc(name string, f interface{}) {
	if hv.fmap == nil {
		hv.fmap = template.FuncMap{}
	}
	hv.fmap[name] = f
}

func (hv *HtmlView) SetViewFuncName(name string) {
	if hv.fmap == nil {
		hv.fmap = template.FuncMap{}
	}

	if _, exists := hv.fmap[name]; !exists {
		hv.fmap[name] = func(i ...interface{}) string { return "" }
	}
}

func (hv *HtmlView) SetGetTmpl(f func(fmap template.FuncMap) (t *template.Template, err error)) {
	hv.GetTmpl = f
}

func (hv *HtmlView) SetVMap(vmap ...mvc.VM) {
	if hv.VM == nil {
		hv.VM = mvc.VM{}
	}
	for _, vm := range vmap {
		for k, v := range vm {
			hv.VM[k] = v
		}
	}
}

func (hv *HtmlView) SetData(name string, value interface{}) {
	if hv.VM == nil {
		hv.VM = mvc.VM{}
	}
	hv.VM[name] = value
}

func (hv *HtmlView) ContentType(ctxt *web.Context) string {
	return "text/html; charset=utf-8"
}

func (hv *HtmlView) getTmpl(names mvc.MvcMeta) (tmpl *template.Template, err error) {
	if hv.GetTmpl != nil {
		tmpl, err = hv.GetTmpl(hv.fmap)
	} else {
		tmpl, err = mvc.GetHtmlTmpl(mvc.GetTemplateRelPath(names, ".html"), hv.fmap)
	}

	return
}

func (hv *HtmlView) Render(ctxt *web.Context) (b []byte, err error) {
	if tmpl, err := hv.getTmpl(mvc.GetMvcMeta(ctxt)); err == nil {
		buf := &bytes.Buffer{}
		tmpl.Execute(buf, hv.VM)
		b = buf.Bytes()
	}

	return
}

func (hv *HtmlView) RenderString(ctxt *web.Context) (s string, err error) {
	var b []byte
	b, err = hv.Render(ctxt)
	if err == nil {
		s = string(b)
	}
	return
}

func (hv *HtmlView) Publish(ctxt *web.Context) (err error) {
	names := mvc.GetMvcMeta(ctxt)
	if names[mvc.MVC_ACTION] == "" {
		names[mvc.MVC_ACTION] = "_"
	}

	var tmpl *template.Template
	tmpl, err = hv.getTmpl(names)

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

			err = tmpl.Execute(tw, hv.VM)

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
