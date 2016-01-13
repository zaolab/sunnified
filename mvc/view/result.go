package view

import (
	"bytes"
	"compress/gzip"
	"errors"
	//"github.com/bradfitz/gomemcache/memcache"
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/util/validate"
	"github.com/zaolab/sunnified/web"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
)

type ResultView struct {
	mvc.VM
	GetTmpl func(fmap template.FuncMap) (t *template.Template, err error)
	fmap    template.FuncMap
}

func (this *ResultView) SetViewFunc(name string, f interface{}) {
	if this.fmap == nil {
		this.fmap = template.FuncMap{}
	}
	this.fmap[name] = f
}

func (this *ResultView) SetViewFuncName(name string) {
	if this.fmap == nil {
		this.fmap = template.FuncMap{}
	}

	if _, exists := this.fmap[name]; !exists {
		this.fmap[name] = func(i ...interface{}) string { return "" }
	}
}

func (this *ResultView) SetGetTmpl(f func(fmap template.FuncMap) (t *template.Template, err error)) {
	this.GetTmpl = f
}

func (this *ResultView) SetVMap(vmap ...mvc.VM) {
	if this.VM == nil {
		this.VM = mvc.VM{}
	}
	for _, vm := range vmap {
		for k, v := range vm {
			this.VM[k] = v
		}
	}
}

func (this *ResultView) SetData(name string, value interface{}) {
	if this.VM == nil {
		this.VM = mvc.VM{}
	}
	this.VM[name] = value
}

func (this *ResultView) ContentType(ctxt *web.Context) string {
	return GetContentType(mvc.GetMvcMeta(ctxt)[mvc.MVC_TYPE])
}

func (this *ResultView) getTmpl(names mvc.MvcMeta) (tmpl *template.Template, ext string, err error) {
	ext = names[mvc.MVC_TYPE]

	if this.GetTmpl != nil {
		tmpl, err = this.GetTmpl(this.fmap)
	} else {
		tmpl, err = mvc.GetHtmlTmpl(mvc.GetTemplateRelPath(names, ext), this.fmap)

		if err != nil && ext != ".html" {
			tmpl, err = mvc.GetHtmlTmpl(mvc.GetTemplateRelPath(names, ".html"), this.fmap)
			ext = ".html"
		}
	}

	return
}

func (this *ResultView) Render(ctxt *web.Context) (b []byte, err error) {
	if tmpl, ext, err := this.getTmpl(mvc.GetMvcMeta(ctxt)); err == nil {
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
		tmpl.Execute(buf, this.VM)
		if jsonp != "" {
			writeJsonpEnd(jsonp, buf)
		}
		b = buf.Bytes()
	}

	return
}

func (this *ResultView) RenderString(ctxt *web.Context) (s string, err error) {
	var b []byte
	b, err = this.Render(ctxt)
	if err == nil {
		s = string(b)
	}
	return
}

func (this *ResultView) Publish(ctxt *web.Context) (err error) {
	names := mvc.GetMvcMeta(ctxt)
	if names[mvc.MVC_ACTION] == "" {
		names[mvc.MVC_ACTION] = "_"
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
	tmpl, ext, err = this.getTmpl(names)

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
			var b *bytes.Buffer = bytes.NewBuffer(make([]byte, 0, 5120))
			var tw io.Writer = io.MultiWriter(ctxt.Response, b)
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

			err = tmpl.Execute(tw, this.VM)

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

			if flushw, ok := ctxt.RootResponse().(http.Flusher); ok {
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
