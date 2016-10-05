package mware

import (
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/util/validate"
	"github.com/zaolab/sunnified/web"
)

func NewHTMLHeadMiddleWare() *HTMLHeadMiddleWare {
	return &HTMLHeadMiddleWare{
		defaultCSS:     make([]string, 0, 1),
		defaultScripts: make([]string, 0, 1),
		cssbatch:       make(map[string][]string),
		scriptbatch:    make(map[string][]string),
	}
}

func HTMLHeadMiddleWareConstructor() MiddleWare {
	return NewHTMLHeadMiddleWare()
}

type HTMLHeadMiddleWare struct {
	BaseMiddleWare
	defaultTitle   string
	defaultCSS     []string
	defaultScripts []string
	cssbatch       map[string][]string
	scriptbatch    map[string][]string
}

func (mw *HTMLHeadMiddleWare) Body(ctxt *web.Context) {
	head := &HTMLHead{
		cssbatch:    mw.cssbatch,
		scriptbatch: mw.scriptbatch,
		css:         make([]string, 0, 1),
		scripts:     make([]string, 0, 1),
		addedcss:    make([]string, 0, 1),
		addedscript: make([]string, 0, 1),
	}
	ctxt.SetResource("htmlhead", head)
	ctxt.SetTitle_ = head.SetTitle

	if mw.defaultTitle != "" {
		head.SetTitle(mw.defaultTitle)
	}
	if mw.defaultCSS != nil && len(mw.defaultCSS) > 0 {
		head.AddCSS(mw.defaultCSS...)
	}
	if mw.defaultScripts != nil && len(mw.defaultScripts) > 0 {
		head.AddScript(mw.defaultScripts...)
	}
}

func (mw *HTMLHeadMiddleWare) View(ctxt *web.Context, vw mvc.View) {
	var head *HTMLHead
	if dview, ok := vw.(mvc.DataView); ok && ctxt.MapResourceValue("htmlhead", &head) == nil && head != nil {
		dview.SetData("Htmlhead_Title", head.Title())
		dview.SetData("Htmlhead_Css", head.CSS())
		dview.SetData("Htmlhead_Scripts", head.Scripts())
	}
}

func (mw *HTMLHeadMiddleWare) AddDefaultCSS(css ...string) {
	mw.defaultCSS = append(mw.defaultCSS, css...)
}

func (mw *HTMLHeadMiddleWare) AddDefaultScript(script ...string) {
	mw.defaultScripts = append(mw.defaultScripts, script...)
}

func (mw *HTMLHeadMiddleWare) CreateCSSBatch(name string, css ...string) {
	if arr, exists := mw.cssbatch[name]; exists {
		mw.cssbatch[name] = append(arr, css...)
	} else {
		newarr := make([]string, len(css))
		copy(newarr, css)
		mw.cssbatch[name] = newarr
	}
}

func (mw *HTMLHeadMiddleWare) CreateScriptBatch(name string, script ...string) {
	if arr, exists := mw.scriptbatch[name]; exists {
		mw.scriptbatch[name] = append(arr, script...)
	} else {
		newarr := make([]string, len(script))
		copy(newarr, script)
		mw.scriptbatch[name] = newarr
	}
}

type HTMLHead struct {
	title       string
	css         []string
	scripts     []string
	cssbatch    map[string][]string
	scriptbatch map[string][]string
	addedcss    []string
	addedscript []string
}

func (h *HTMLHead) Title() string {
	return h.title
}

func (h *HTMLHead) CSS() []string {
	return h.css
}

func (h *HTMLHead) Scripts() []string {
	return h.scripts
}

func (h *HTMLHead) SetTitle(title string) {
	h.title = title
}

func (h *HTMLHead) AddCSS(css ...string) {
	h.css = append(h.css, css...)
}

func (h *HTMLHead) AddScript(script ...string) {
	h.scripts = append(h.scripts, script...)
}

func (h *HTMLHead) AddCSSBatch(name string) {
	if arr, exists := h.cssbatch[name]; exists && !validate.IsIn(name, h.addedcss...) {
		h.AddCSS(arr...)
		h.addedcss = append(h.addedcss, name)
	}
}

func (h *HTMLHead) AddScriptBatch(name string) {
	if arr, exists := h.scriptbatch[name]; exists && !validate.IsIn(name, h.addedscript...) {
		h.AddScript(arr...)
		h.addedscript = append(h.addedscript, name)
	}
}
