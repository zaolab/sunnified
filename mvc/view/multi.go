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

type MultiView struct {
	mvc.VM
	GetTmpl func(fmap template.FuncMap) (t *template.Template, err error)
	fmap    template.FuncMap
}

func (mv *MultiView) SetViewFunc(name string, f interface{}) {
	if mv.fmap == nil {
		mv.fmap = template.FuncMap{}
	}
	mv.fmap[name] = f
}

func (mv *MultiView) SetViewFuncName(name string) {
	if mv.fmap == nil {
		mv.fmap = template.FuncMap{}
	}

	if _, exists := mv.fmap[name]; !exists {
		mv.fmap[name] = func(i ...interface{}) string { return "" }
	}
}

func (mv *MultiView) SetGetTmpl(f func(fmap template.FuncMap) (t *template.Template, err error)) {
	mv.GetTmpl = f
}

func (mv *MultiView) SetVMap(vmap ...mvc.VM) {
	if mv.VM == nil {
		mv.VM = mvc.VM{}
	}
	for _, vm := range vmap {
		for k, v := range vm {
			mv.VM[k] = v
		}
	}
}

func (mv *MultiView) SetData(name string, value interface{}) {
	if mv.VM == nil {
		mv.VM = mvc.VM{}
	}
	mv.VM[name] = value
}

func (mv *MultiView) ContentType(ctxt *web.Context) string {
	return "text/html; charset=utf-8"
}

func (mv *MultiView) getTmpl(names mvc.MvcMeta) (tmpl *template.Template, ext string, err error) {
	if mv.GetTmpl != nil {
		tmpl, err = mv.GetTmpl(mv.fmap)
	} else {
		tmpl, err = mvc.GetHtmlTmpl(mvc.GetTemplateRelPath(names, ext), mv.fmap)
	}

	return
}

func (mv *MultiView) Render(ctxt *web.Context) (b []byte, err error) {
	if tmpl, _, err := mv.getTmpl(mvc.GetMvcMeta(ctxt)); err == nil {
		buf := &bytes.Buffer{}
		tmpl.Execute(buf, mv.VM)
		b = buf.Bytes()
	}

	return
}

func (mv *MultiView) RenderString(ctxt *web.Context) (s string, err error) {
	var b []byte
	b, err = mv.Render(ctxt)
	if err == nil {
		s = string(b)
	}
	return
}

func (mv *MultiView) Publish(ctxt *web.Context) (err error) {
	names := mvc.GetMvcMeta(ctxt)
	if names[mvc.MVC_ACTION] == "" {
		names[mvc.MVC_ACTION] = "_"
	}

	var tmpl *template.Template
	tmpl, _, err = mv.getTmpl(names)

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

			err = tmpl.Execute(tw, mv.VM)

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
