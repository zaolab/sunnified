package router

import (
	"errors"
	"github.com/zaolab/sunnified/web"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var ErrInvalidHandler = errors.New("Invalid Handler")

const (
	_ = iota
	ROUTE_HARD
	ROUTE_REGEX
	ROUTE_TYPED
	ROUTE_SOFT
)

const (
	MATCHTYPE_INT = iota
	MATCHTYPE_INT64
	MATCHTYPE_FLOAT
	MATCHTYPE_FLOAT64
)

type ContextHTTPHandler struct {
	web.ContextHandler
}

func (this ContextHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	this.ContextHandler.ServeContextHTTP(web.NewContext(w, r))
}

type ContextHTTPHandlerFunc func(*web.Context)

func (f ContextHTTPHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(web.NewContext(w, r))
}

func (f ContextHTTPHandlerFunc) ServeContextHTTP(ctxt *web.Context) {
	f(ctxt)
}

func NewSunnyRoute() *SunnyRoute {
	return &SunnyRoute{
		hardroute: map[string]Route{},
		regxroute: map[*regexp.Regexp]Route{},
		typeroute: [4]Route{},
	}
}

type SunnySwitch struct {
	Route
	varnames []string
}

func (this *SunnySwitch) Handle(p string, h http.Handler) (ep EndPoint) {
	ep = this.Route.Handle(p, h)
	ep.PrependVarName(this.varnames...)
	return
}

func (this *SunnySwitch) Switch(p string) (s Switch) {
	s = this.Route.Switch(p)
	s.PrependVarName(this.varnames...)
	return
}

func (this *SunnySwitch) PrependVarName(names ...string) {
	var lnames = len(names)
	if lnames == 0 {
		return
	}

	var varnames = make([]string, len(this.varnames)+lnames)

	copy(varnames[0:lnames], names)

	if lvnames := len(varnames); lvnames > lnames {
		copy(varnames[lnames:lvnames], this.varnames)
	}

	this.varnames = varnames
}

type SunnyRoute struct {
	hardend EndPoint
	softend EndPoint

	hardroute map[string]Route
	regxroute map[*regexp.Regexp]Route
	typeroute [4]Route
	softroute Route
}

func (this *SunnyRoute) Switch(p string) Switch {
	varnames, r := this.BuildRoute(p)
	if len(r) > 0 {
		return &SunnySwitch{
			varnames: varnames[len(varnames)-1],
			Route:    r[len(r)-1],
		}
	}
	return nil
}

func (this *SunnyRoute) BuildRoute(p string) (varnames [][]string, rts []Route) {
	p = strings.TrimSpace(p)
	p = strings.TrimLeft(p, "/")

	if p == "" {
		rts = []Route{this}
		varnames = [][]string{[]string{}}
		return
	}

	var (
		pathsplit = strings.SplitN(p, "/", 2)
		curpath   = pathsplit[0]
		exists    bool
		varname   string
		rt        Route
		addroute  bool
	)

	if curpath[0] == '{' && curpath[len(curpath)-1] == '}' {
		curpath = strings.TrimSpace(curpath[1 : len(pathsplit[0])-1])
		varname = curpath

		if strings.Contains(curpath, ":") {
			curpathsplit := strings.SplitN(curpath, ":", 2)

			switch strings.ToLower(curpathsplit[1]) {
			case "int32":
				fallthrough
			case "int":
				if rt = this.typeroute[MATCHTYPE_INT]; rt == nil {
					rt = NewSunnyRoute()
					this.typeroute[MATCHTYPE_INT] = rt
				}

			case "int64":
				if rt = this.typeroute[MATCHTYPE_INT64]; rt == nil {
					rt = NewSunnyRoute()
					this.typeroute[MATCHTYPE_INT64] = rt
				}

			case "float32":
				fallthrough
			case "float":
				if rt = this.typeroute[MATCHTYPE_FLOAT]; rt == nil {
					rt = NewSunnyRoute()
					this.typeroute[MATCHTYPE_FLOAT] = rt
				}

			case "float64":
				if rt = this.typeroute[MATCHTYPE_FLOAT64]; rt == nil {
					rt = NewSunnyRoute()
					this.typeroute[MATCHTYPE_FLOAT64] = rt
				}

			default:
				for re, rr := range this.regxroute {
					if re.String() == curpathsplit[1] {
						rt = rr
						break
					}
				}

				if rt == nil {
					rt = NewSunnyRoute()
					regexstr := curpathsplit[1]
					if regexstr[0] != '^' {
						regexstr = "^" + regexstr
					}
					if regexstr[len(regexstr)-1] != '$' {
						regexstr = regexstr + "$"
					}
					this.regxroute[regexp.MustCompile(regexstr)] = rt
				}
			}

			varname = strings.TrimSpace(curpathsplit[0])
		} else {
			if rt = this.softroute; rt == nil {
				rt = NewSunnyRoute()
				this.softroute = rt
			}
		}

		if varname != "" && varname[len(varname)-1] == '*' {
			varname = strings.TrimSpace(varname[0 : len(varname)-1])
			addroute = true
		}

		if varname == "" {
			varname = "_"
		}
	} else {
		if rt, exists = this.hardroute[pathsplit[0]]; !exists {
			rt = NewSunnyRoute()
			this.hardroute[pathsplit[0]] = rt
		}
	}

	if lpathsplit := len(pathsplit); lpathsplit <= 1 || pathsplit[1] == "" {
		varnames, rts = rt.BuildRoute("")
	} else {
		varnames, rts = rt.BuildRoute(pathsplit[1])
	}

	if varname != "" {
		for i, vnames := range varnames {
			tvarnames := make([]string, len(vnames)+1)
			tvarnames[0] = varname
			copy(tvarnames[1:len(tvarnames)], vnames)
			varnames[i] = tvarnames
		}
	}

	if addroute {
		trts := make([]Route, len(rts)+1)
		trts[0] = this
		copy(trts[1:len(trts)], rts)
		rts = trts

		tvarnames := make([][]string, len(varnames)+1)
		tvarnames[0] = []string{varname}
		copy(tvarnames[1:len(tvarnames)], varnames)
		varnames = tvarnames
	}

	return
}

func (this *SunnyRoute) HasRoute(p string) bool {
	ep, _, _ := this.FindEndPoint(this.SplitPath(p), make([]string, 0, 3))
	return ep != nil
}

func (this *SunnyRoute) SplitPath(p string) (pathsplit []string) {
	pathsplit = strings.Split(p, "/")
	lpath := len(pathsplit)
	if lpath > 0 && pathsplit[0] == "" {
		pathsplit = pathsplit[1:lpath]
		lpath--
	}
	if lpath > 0 && pathsplit[lpath-1] == "" {
		pathsplit = pathsplit[0 : lpath-1]
	}
	return
}

func (this *SunnyRoute) Handle(p string, h interface{}, method ...string) (ep EndPoint) {
	p = strings.TrimSpace(p)

	if p == "" {
		ep = &SunnyEndPoint{}
		ep.SetHandler(h, method...)
		this.hardend = ep
	} else if p == "/" {
		ep = &SunnyEndPoint{}
		ep.SetHandler(h, method...)
		this.softend = ep
	} else {
		varnames, rts := this.BuildRoute(p)

		if len(rts) == 1 {
			rt := rts[0]
			if p[len(p)-1] == '/' {
				ep = rt.Handle("/", h, method...)
			} else {
				ep = rt.Handle("", h, method...)
			}

			if len(varnames) > 0 {
				ep.PrependVarName(varnames[0]...)
			}
		} else {
			for i, rt := range rts {
				if p[len(p)-1] == '/' {
					ep = rt.Handle("/", h, method...)
				} else {
					ep = rt.Handle("", h, method...)
				}

				ep.PrependVarName(varnames[i]...)
			}
		}
	}

	return
}

func (this *SunnyRoute) FindEndPoint(p []string, data []string) (EndPoint, []string, []string) {
	var lpath = len(p)

	if data == nil {
		data = make([]string, 0, 3)
	}

	if lpath <= 0 {
		if this.hardend != nil {
			return this.hardend, p, data
		} else {
			return this.softend, p, data
		}
	}

	var curpath = p[0]
	var noext = ""
	var fullp = p
	p = p[1:lpath]

	if lpath == 1 && path.Ext(curpath) != "" {
		noext = strings.TrimSuffix(curpath, path.Ext(curpath))
	}

	if route, exists := this.hardroute[curpath]; exists {
		return route.FindEndPoint(p, data)
	} else if noext != "" {
		if route, exists := this.hardroute[curpath]; exists {
			return route.FindEndPoint(p, data)
		}
	}

	for regx, route := range this.regxroute {
		if noext != "" && regx.MatchString(noext) {
			return nextEndPoint(route, noext, p, data)
		} else if regx.MatchString(curpath) {
			return nextEndPoint(route, curpath, p, data)
		}
	}

	typepath := noext
	if typepath == "" {
		typepath = curpath
	}

	for rtype, route := range this.typeroute {
		if route == nil {
			continue
		}
		switch rtype {
		case MATCHTYPE_INT:
			if _, err := strconv.Atoi(typepath); err == nil {
				return nextEndPoint(route, typepath, p, data)
			}
		case MATCHTYPE_INT64:
			if _, err := strconv.ParseInt(typepath, 10, 0); err == nil {
				return nextEndPoint(route, typepath, p, data)
			}
		case MATCHTYPE_FLOAT:
			if _, err := strconv.ParseFloat(typepath, 32); err == nil {
				return nextEndPoint(route, typepath, p, data)
			}
		case MATCHTYPE_FLOAT64:
			if _, err := strconv.ParseFloat(typepath, 64); err == nil {
				return nextEndPoint(route, typepath, p, data)
			}
		}
	}

	if this.softroute != nil {
		return nextEndPoint(this.softroute, typepath, p, data)
	}

	p = fullp
	return this.softend, p, data
}

func (this *SunnyRoute) FindRequestedEndPoint(p string, r *http.Request) *RequestedEndPoint {
	pathsplit := this.SplitPath(p)
	ep, upath, data := this.FindEndPoint(pathsplit, make([]string, 0, 3))
	if ep != nil {
		return ep.GetRequestedEndPoint(r, upath, data)
	}
	return nil
}

func (this *SunnyRoute) HardEndPoint() EndPoint {
	return this.hardend
}

func (this *SunnyRoute) SoftEndPoint() EndPoint {
	return this.softend
}

type SunnyEndPoint struct {
	get    http.Handler
	post   http.Handler
	put    http.Handler
	patch  http.Handler
	delete http.Handler
	head   http.Handler

	varnames []string
}

func (this *SunnyEndPoint) PrependVarName(names ...string) {
	var lnames = len(names)
	if lnames == 0 {
		return
	}

	var varnames = make([]string, len(this.varnames)+lnames)

	copy(varnames[0:lnames], names)

	if lvnames := len(varnames); lvnames > lnames {
		copy(varnames[lnames:lvnames], this.varnames)
	}

	this.varnames = varnames
}

func (this *SunnyEndPoint) AppendVarName(name ...string) {
	this.varnames = append(this.varnames, name...)
}

func (this *SunnyEndPoint) SetHandler(handler interface{}, method ...string) error {
	var (
		h  http.Handler
		ch web.ContextHandler
		hf func(w http.ResponseWriter, r *http.Request)
		cf func(*web.Context)
		ok bool
	)

	if h, ok = handler.(http.Handler); !ok {
		if hf, ok = handler.(func(w http.ResponseWriter, r *http.Request)); ok {
			h = http.HandlerFunc(hf)
		} else if ch, ok = handler.(web.ContextHandler); ok {
			h = ContextHTTPHandler{ch}
		} else if cf, ok = handler.(func(*web.Context)); ok {
			h = ContextHTTPHandlerFunc(cf)
		} else if handler != nil {
			return ErrInvalidHandler
		}
	}

	if len(method) == 0 {
		this.get = h
		this.post = h
		this.put = h
		this.patch = h
		this.delete = h
	} else {
		for _, m := range method {
			switch strings.ToUpper(m) {
			case "GET":
				this.get = h
			case "POST":
				this.post = h
			case "PUT":
				this.put = h
			case "PATCH":
				this.patch = h
			case "DELETE":
				this.delete = h
			case "HEAD":
				this.head = h
			}
		}
	}

	return nil
}

func (this *SunnyEndPoint) Handlers() map[string]http.Handler {
	return map[string]http.Handler{
		"GET":    this.get,
		"POST":   this.post,
		"PUT":    this.put,
		"PATCH":  this.patch,
		"DELETE": this.delete,
		"HEAD":   this.HeadHandler(),
	}
}

func (this *SunnyEndPoint) GetHandler() http.Handler {
	return this.get
}

func (this *SunnyEndPoint) PostHandler() http.Handler {
	return this.post
}

func (this *SunnyEndPoint) PutHandler() http.Handler {
	return this.put
}

func (this *SunnyEndPoint) PatchHandler() http.Handler {
	return this.patch
}

func (this *SunnyEndPoint) DeleteHandler() http.Handler {
	return this.delete
}

func (this *SunnyEndPoint) HeadHandler() (head http.Handler) {
	if head = this.head; head == nil && this.get != nil {
		head = this.get
	}
	return
}

func (this *SunnyEndPoint) Handler(method string) http.Handler {
	method = strings.ToUpper(method)
	return this.Handlers()[method]
}

func (this *SunnyEndPoint) GetRequestedEndPoint(r *http.Request, upath []string, data []string) *RequestedEndPoint {
	handlers := this.Handlers()
	if h, exists := handlers[r.Method]; exists && h != nil {
		var pdata = make(map[string]string)
		for i, value := range data {
			if this.varnames[i] != "_" {
				pdata[this.varnames[i]] = value
			}
		}

		ext := ""
		lastele := len(upath) - 1

		if lastele >= 0 && strings.Contains(upath[lastele], ".") {
			ext = path.Ext(upath[lastele])
			upath[lastele] = strings.TrimSuffix(upath[lastele], ext)
		} else {
			ext = path.Ext(strings.TrimRight(r.URL.Path, "/"))
		}

		return &RequestedEndPoint{
			Ext:      ext,
			UPath:    web.UPath(upath),
			PData:    web.PData(pdata),
			Method:   r.Method,
			Handler:  h,
			EndPoint: this,
		}
	}

	return nil
}

func (this *SunnyEndPoint) Methods() (methods []string) {
	methods = make([]string, 0, 6)
	handlers := this.Handlers()

	for meth, h := range handlers {
		if h != nil {
			methods = append(methods, meth)
		}
	}

	return
}

func nextEndPoint(route Route, curpath string, p []string, data []string) (EndPoint, []string, []string) {
	data = append(data, curpath)
	return route.FindEndPoint(p, data)
}
