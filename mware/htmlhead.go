package mware

import (
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/util/validate"
	"github.com/zaolab/sunnified/web"
)

func NewHTMLHeadMiddleWare() *HTMLHeadMiddleWare {
	return &HTMLHeadMiddleWare{
		defaultCss:     make([]string, 0, 1),
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
	defaultCss     []string
	defaultScripts []string
	cssbatch       map[string][]string
	scriptbatch    map[string][]string
}

func (this *HTMLHeadMiddleWare) Body(ctxt *web.Context) {
	head := &HTMLHead{
		cssbatch:    this.cssbatch,
		scriptbatch: this.scriptbatch,
		css:         make([]string, 0, 1),
		scripts:     make([]string, 0, 1),
		addedcss:    make([]string, 0, 1),
		addedscript: make([]string, 0, 1),
	}
	ctxt.SetResource("htmlhead", head)
	ctxt.SetTitle_ = head.SetTitle

	if this.defaultTitle != "" {
		head.SetTitle(this.defaultTitle)
	}
	if this.defaultCss != nil && len(this.defaultCss) > 0 {
		head.AddCss(this.defaultCss...)
	}
	if this.defaultScripts != nil && len(this.defaultScripts) > 0 {
		head.AddScript(this.defaultScripts...)
	}
}

func (this *HTMLHeadMiddleWare) View(ctxt *web.Context, vw mvc.View) {
	var head *HTMLHead
	if dview, ok := vw.(mvc.DataView); ok && ctxt.MapResourceValue("htmlhead", &head) == nil && head != nil {
		dview.SetData("Htmlhead_Title", head.Title())
		dview.SetData("Htmlhead_Css", head.Css())
		dview.SetData("Htmlhead_Scripts", head.Scripts())
	}
}

func (this *HTMLHeadMiddleWare) AddDefaultCss(css ...string) {
	this.defaultCss = append(this.defaultCss, css...)
}

func (this *HTMLHeadMiddleWare) AddDefaultScript(script ...string) {
	this.defaultScripts = append(this.defaultScripts, script...)
}

func (this *HTMLHeadMiddleWare) CreateCssBatch(name string, css ...string) {
	if arr, exists := this.cssbatch[name]; exists {
		this.cssbatch[name] = append(arr, css...)
	} else {
		newarr := make([]string, len(css))
		copy(newarr, css)
		this.cssbatch[name] = newarr
	}
}

func (this *HTMLHeadMiddleWare) CreateScriptBatch(name string, script ...string) {
	if arr, exists := this.scriptbatch[name]; exists {
		this.scriptbatch[name] = append(arr, script...)
	} else {
		newarr := make([]string, len(script))
		copy(newarr, script)
		this.scriptbatch[name] = newarr
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

func (this *HTMLHead) Title() string {
	return this.title
}

func (this *HTMLHead) Css() []string {
	return this.css
}

func (this *HTMLHead) Scripts() []string {
	return this.scripts
}

func (this *HTMLHead) SetTitle(title string) {
	this.title = title
}

func (this *HTMLHead) AddCss(css ...string) {
	this.css = append(this.css, css...)
}

func (this *HTMLHead) AddScript(script ...string) {
	this.scripts = append(this.scripts, script...)
}

func (this *HTMLHead) AddCssBatch(name string) {
	if arr, exists := this.cssbatch[name]; exists && !validate.IsIn(name, this.addedcss...) {
		this.AddCss(arr...)
		this.addedcss = append(this.addedcss, name)
	}
}

func (this *HTMLHead) AddScriptBatch(name string) {
	if arr, exists := this.scriptbatch[name]; exists && !validate.IsIn(name, this.addedscript...) {
		this.AddScript(arr...)
		this.addedscript = append(this.addedscript, name)
	}
}
