package mvc

import (
	"fmt"
	"html/template"
	"net/url"
	"path/filepath"
	"regexp"
	"sync"
	txtemplate "text/template"
	"time"

	//_ "github.com/zaolab/sunnified/util/extype"
	"github.com/zaolab/sunnified/web"
)

var (
	htmplcache    = make(map[string]*htcachedetails)
	hmutex        = sync.RWMutex{}
	ttmplcache    = make(map[string]*ttcachedetails)
	tmutex        = sync.RWMutex{}
	CacheDuration = time.Minute * 1 / 60
	fmutex        = sync.RWMutex{}
	funcnames     = make(template.FuncMap)
	urlreplchars  = regexp.MustCompile(`[\s\.,]`)
	urldumpchars  = regexp.MustCompile(`['"&@#%=<>:;\/\\\$\^\+\|\[\]\?\{\}]`)
)

func AddFuncName(names ...string) {
	fmutex.Lock()
	defer fmutex.Unlock()
	for _, name := range names {
		funcnames[name] = emptyf
	}
}

func GetFuncMap() (fc template.FuncMap) {
	fmutex.RLock()
	defer fmutex.RUnlock()
	fc = make(template.FuncMap)
	for k, v := range funcnames {
		fc[k] = v
	}
	return
}

func GetTextFuncMap() (fc txtemplate.FuncMap) {
	return txtemplate.FuncMap(GetFuncMap())
}

func emptyf(s ...interface{}) string {
	return ""
}

func GetTemplateRelPath(names Meta, ext string) string {
	return fmt.Sprintf("themes/default/tmpl/%s/%s/%s%s", names[MVCModule], names[MVCController], names[MVCAction], ext)
}

type View interface {
	ContentType(*web.Context) string
	Render(*web.Context) ([]byte, error)
	RenderString(*web.Context) (string, error)
	Publish(*web.Context) error
}

type TmplView interface {
	SetViewFunc(string, interface{})
	SetViewFuncName(string)
}

type HTMLTmplView interface {
	TmplView
	SetGetTmpl(func(fmap template.FuncMap) (t *template.Template, err error))
}

type TextTmplView interface {
	TmplView
	SetGetTmpl(func(fmap txtemplate.FuncMap) (t *txtemplate.Template, err error))
}

type DataView interface {
	SetVMap(...VM)
	SetData(string, interface{})
}

type htcachedetails struct {
	cdown *time.Timer
	t     *template.Template
}

type ttcachedetails struct {
	cdown *time.Timer
	t     *txtemplate.Template
}

func GetHTMLTmpl(p string, fmap template.FuncMap) (t *template.Template, err error) {
	if t = getHTMLCache(p); t == nil {
		ap, e := filepath.Abs(p)
		if e != nil {
			ap = p
		}
		t = template.New(filepath.Base(ap))
		t = t.Funcs(GetFuncMap())
		t, err = t.ParseFiles(ap)

		if err == nil {
			// TODO: remove hardcoded directory structure
			sharep, e := filepath.Abs("themes/default/tmpl/_share_/*" + filepath.Ext(ap))
			if e == nil {
				t.ParseGlob(sharep)
			}
		} else {
			panic(err)
		}

		setHTMLCache(p, t)
		t, _ = t.Clone()
	}

	t = t.Funcs(fmap)
	return
}

func GetTextTmpl(p string, fmap txtemplate.FuncMap) (t *txtemplate.Template, err error) {
	if t = getTextCache(p); t == nil {
		ap, e := filepath.Abs(p)
		if e != nil {
			ap = p
		}
		t = txtemplate.New(filepath.Base(ap))
		t = t.Funcs(GetTextFuncMap())
		t, err = t.ParseFiles(ap)

		if err == nil {
			// TODO: remove hardcoded directory structure
			sharep, e := filepath.Abs("themes/default/tmpl/_share_/*" + filepath.Ext(ap))
			if e == nil {
				t.ParseGlob(sharep)
			}
		} else {
			panic(err)
		}

		setTextCache(p, t)
		t, _ = t.Clone()
	}

	t = t.Funcs(fmap)
	return
}

func getHTMLCache(p string) *template.Template {
	hmutex.RLock()
	defer hmutex.RUnlock()
	if _, ok := htmplcache[p]; ok {
		htmplcache[p].cdown.Reset(CacheDuration)
		t, _ := htmplcache[p].t.Clone()
		return t
	}
	return nil
}

func setHTMLCache(p string, t *template.Template) {
	hmutex.Lock()
	defer hmutex.Unlock()
	if _, ok := htmplcache[p]; ok {
		htmplcache[p].cdown.Stop()
		htmplcache[p].t = nil
	}

	timer := time.AfterFunc(CacheDuration, func() { delHTMLCache(p) })
	htmplcache[p] = &htcachedetails{timer, t}
}

func delHTMLCache(p string) {
	hmutex.Lock()
	defer hmutex.Unlock()
	if _, ok := htmplcache[p]; ok {
		htmplcache[p].cdown.Stop()
		htmplcache[p].t = nil
		delete(htmplcache, p)
	}
}

func getTextCache(p string) *txtemplate.Template {
	tmutex.RLock()
	defer tmutex.RUnlock()
	if _, ok := ttmplcache[p]; ok {
		ttmplcache[p].cdown.Reset(CacheDuration)
		t, _ := ttmplcache[p].t.Clone()
		return t
	}
	return nil
}

func setTextCache(p string, t *txtemplate.Template) {
	tmutex.Lock()
	defer tmutex.Unlock()
	if _, ok := ttmplcache[p]; ok {
		ttmplcache[p].cdown.Stop()
		ttmplcache[p].t = nil
	}

	timer := time.AfterFunc(CacheDuration, func() { delTextCache(p) })
	ttmplcache[p] = &ttcachedetails{timer, t}
}

func delTextCache(p string) {
	tmutex.Lock()
	defer tmutex.Unlock()
	if _, ok := ttmplcache[p]; ok {
		ttmplcache[p].cdown.Stop()
		ttmplcache[p].t = nil
		delete(ttmplcache, p)
	}
}

func URLSafeTrashString(s string) string {
	s = urlreplchars.ReplaceAllLiteralString(s, "-")
	s = urldumpchars.ReplaceAllLiteralString(s, "")
	return url.QueryEscape(s)
}
