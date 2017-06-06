package view

import (
	"bytes"
	"compress/gzip"
	"errors"
	"html/template"
	"io"
	"log"
	"mime"
	"strings"

	//"github.com/bradfitz/gomemcache/memcache"
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/util/validate"
	"github.com/zaolab/sunnified/web"
)

type ResultView struct {
	mvc.VM
	GetTmpl func(fmap template.FuncMap) (t *template.Template, err error)
	fmap    template.FuncMap
}

func (rv *ResultView) SetViewFunc(name string, f interface{}) {
	if rv.fmap == nil {
		rv.fmap = template.FuncMap{}
	}
	rv.fmap[name] = f
}

func (rv *ResultView) SetViewFuncName(name string) {
	if rv.fmap == nil {
		rv.fmap = template.FuncMap{}
	}

	if _, exists := rv.fmap[name]; !exists {
		rv.fmap[name] = func(i ...interface{}) string { return "" }
	}
}

func (rv *ResultView) SetGetTmpl(f func(fmap template.FuncMap) (t *template.Template, err error)) {
	rv.GetTmpl = f
}

func (rv *ResultView) SetVMap(vmap ...mvc.VM) {
	if rv.VM == nil {
		rv.VM = mvc.VM{}
	}
	for _, vm := range vmap {
		for k, v := range vm {
			rv.VM[k] = v
		}
	}
}

func (rv *ResultView) SetData(name string, value interface{}) {
	if rv.VM == nil {
		rv.VM = mvc.VM{}
	}
	rv.VM[name] = value
}

func (rv *ResultView) ContentType(ctxt *web.Context) string {
	return GetContentType(mvc.GetMvcMeta(ctxt)[mvc.MVCType])
}

func (rv *ResultView) getTmpl(names mvc.Meta) (tmpl *template.Template, ext string, err error) {
	ext = names[mvc.MVCType]

	if rv.GetTmpl != nil {
		tmpl, err = rv.GetTmpl(rv.fmap)
	} else {
		tmpl, err = mvc.GetHTMLTmpl(mvc.GetTemplateRelPath(names, ext), rv.fmap)

		if err != nil && ext != ".html" {
			tmpl, err = mvc.GetHTMLTmpl(mvc.GetTemplateRelPath(names, ".html"), rv.fmap)
			ext = ".html"
		}
	}

	return
}

func (rv *ResultView) Render(ctxt *web.Context) (b []byte, err error) {
	if tmpl, ext, err := rv.getTmpl(mvc.GetMvcMeta(ctxt)); err == nil {
		buf := &bytes.Buffer{}
		var jsonp string
		if ext == ".jsonp" {
			jsonp = ctxt.RequestValue("callback")
			if jsonp == "" {
				jsonp = ctxt.RequestValue("jsonp")
			}
			jsonp = strings.TrimSpace(jsonp)
		}
		if jsonp != "" {
			writeJsonpStart(jsonp, buf)
		}
		tmpl.Execute(buf, rv.VM)
		if jsonp != "" {
			writeJsonpEnd(jsonp, buf)
		}
		b = buf.Bytes()
	}

	return
}

func (rv *ResultView) RenderString(ctxt *web.Context) (s string, err error) {
	var b []byte
	b, err = rv.Render(ctxt)
	if err == nil {
		s = string(b)
	}
	return
}

func (rv *ResultView) Publish(ctxt *web.Context) (err error) {
	names := mvc.GetMvcMeta(ctxt)
	if names[mvc.MVCAction] == "" {
		names[mvc.MVCAction] = "_"
	}

	var tmpl *template.Template
	var ext string

	/*
		var mc = memcache.New("127.0.0.1:11211")
		var item *memcache.Item
		if item, err = mc.Get(ctxt.Request.RequestURI); err == nil {
			ctxt.SetHeader("Content-Type", GetContentType(ext))
			if ctxt.ReqHeaderHas("Accept-Encoding", "gzip") {
				ctxt.SetHeader("Content-Encoding", "gzip")
			}
			ctxt.Response.Write(item.Value)
			return
		}
	*/
	tmpl, ext, err = rv.getTmpl(names)

	if err == nil {
		var isjsonp bool
		var jsonp string
		var method = ctxt.Method()

		if ext == ".jsonp" {
			jsonp = ctxt.RequestValue("callback")
			if jsonp == "" {
				jsonp = ctxt.RequestValue("jsonp")
			}

			if (method == "GET" || method == "HEAD") && jsonp != "" && ext == ".jsonp" && validate.IsJSONPCallback(jsonp) {
				ctxt.SetHeader("Content-Type", "application/javascript")
				ctxt.SetHeader("Content-Disposition", "attachment; filename=jsonp.jsonp")
				ctxt.SetHeader("X-Content-Type-Options", "nosniff")
				isjsonp = true
			} else {
				err = errors.New("Invalid jsonp callback")
				log.Println(err)
				ctxt.SetErrorCode(403)
				return
			}
		} else {
			ctxt.SetHeader("Content-Type", GetContentType(ext))
		}

		if method != "HEAD" {
			var err error
			var b = bytes.NewBuffer(make([]byte, 0, 5120))
			var tw = io.MultiWriter(ctxt.Response, b)
			var gzipwriter *gzip.Writer

			if ctxt.ReqHeaderHas("Accept-Encoding", "gzip") {
				ctxt.SetHeader("Content-Encoding", "gzip")
				gzipwriter, _ = gzip.NewWriterLevel(tw, gzip.BestSpeed)
				tw = gzipwriter
			}

			ctxt.SetHeader("Vary", "Accept-Encoding")
			ctxt.Response.WriteHeader(200)

			if isjsonp {
				writeJsonpStart(jsonp, tw)
			}

			err = tmpl.Execute(tw, rv.VM)

			if err != nil {
				// Header already sent... multiple write headers
				//panic(err)
				log.Println(err)
			}

			if isjsonp {
				writeJsonpEnd(jsonp, tw)
			}

			if gzipwriter != nil {
				gzipwriter.Close()
			}

			//mc.Set(&memcache.Item{Key: ctxt.Request.RequestURI, Value: b.Bytes(), Expiration: 3600})

			//if flushw, ok := ctxt.RootResponse().(http.Flusher); ok {
			//	flushw.Flush()
			//}
		} else {
			ctxt.Response.WriteHeader(200)
		}
	} else {
		log.Println(err)
		ctxt.SetErrorCode(500)
	}

	return
}

func writeJsonpStart(jsonp string, w io.Writer) {
	w.Write([]byte{'t', 'y', 'p', 'e', 'o', 'f', ' '})
	w.Write([]byte(jsonp))
	w.Write([]byte{'=', '=', '=', '"', 'f', 'u', 'n', 'c', 't', 'i', 'o', 'n', '"', ' ', '&', '&', ' '})
	w.Write([]byte(jsonp))
	w.Write([]byte{'('})
}

func writeJsonpEnd(jsonp string, w io.Writer) {
	w.Write([]byte{')'})
}

func NewResultView(vmap mvc.VM) *ResultView {
	if vmap == nil {
		vmap = mvc.VM{}
	}
	return &ResultView{VM: vmap, fmap: template.FuncMap{}}
}

func GetContentType(ext string) (contentType string) {
	contentType = mime.TypeByExtension(ext)

	if contentType == "" {
		// extension's mime type missing from mime package, should import _ "sunnified/util/extype"
		// for list of mime excluded from the mime package
		if ext == ".json" {
			contentType = "application/json; charset=utf-8"
		} else {
			contentType = "text/html; charset=utf-8"
		}
	}

	return
}
