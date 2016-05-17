package router

import (
	"errors"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/zaolab/sunnified/web"
)

var ErrInvalidHandler = errors.New("invalid handler")

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

func (ch ContextHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ch.ContextHandler.ServeContextHTTP(web.NewContext(w, r))
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

func (ss *SunnySwitch) Handle(p string, h http.Handler) (ep EndPoint) {
	ep = ss.Route.Handle(p, h)
	ep.PrependVarName(ss.varnames...)
	return
}

func (ss *SunnySwitch) Switch(p string) (s Switch) {
	s = ss.Route.Switch(p)
	s.PrependVarName(ss.varnames...)
	return
}

func (ss *SunnySwitch) PrependVarName(names ...string) {
	var lnames = len(names)
	if lnames == 0 {
		return
	}

	var varnames = make([]string, len(ss.varnames)+lnames)

	copy(varnames[0:lnames], names)

	if lvnames := len(varnames); lvnames > lnames {
		copy(varnames[lnames:lvnames], ss.varnames)
	}

	ss.varnames = varnames
}

type SunnyRoute struct {
	hardend EndPoint
	softend EndPoint

	hardroute map[string]Route
	regxroute map[*regexp.Regexp]Route
	typeroute [4]Route
	softroute Route
}

func (sr *SunnyRoute) Switch(p string) Switch {
	varnames, r := sr.BuildRoute(p)
	if len(r) > 0 {
		return &SunnySwitch{
			varnames: varnames[len(varnames)-1],
			Route:    r[len(r)-1],
		}
	}
	return nil
}

func (sr *SunnyRoute) BuildRoute(p string) (varnames [][]string, rts []Route) {
	p = strings.TrimSpace(p)
	p = strings.TrimLeft(p, "/")

	if p == "" {
		rts = []Route{sr}
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
			case "int32", "int":
				if rt = sr.typeroute[MATCHTYPE_INT]; rt == nil {
					rt = NewSunnyRoute()
					sr.typeroute[MATCHTYPE_INT] = rt
				}

			case "int64":
				if rt = sr.typeroute[MATCHTYPE_INT64]; rt == nil {
					rt = NewSunnyRoute()
					sr.typeroute[MATCHTYPE_INT64] = rt
				}

			case "float32", "float":
				if rt = sr.typeroute[MATCHTYPE_FLOAT]; rt == nil {
					rt = NewSunnyRoute()
					sr.typeroute[MATCHTYPE_FLOAT] = rt
				}

			case "float64":
				if rt = sr.typeroute[MATCHTYPE_FLOAT64]; rt == nil {
					rt = NewSunnyRoute()
					sr.typeroute[MATCHTYPE_FLOAT64] = rt
				}

			default:
				for re, rr := range sr.regxroute {
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
					sr.regxroute[regexp.MustCompile(regexstr)] = rt
				}
			}

			varname = strings.TrimSpace(curpathsplit[0])
		} else {
			if rt = sr.softroute; rt == nil {
				rt = NewSunnyRoute()
				sr.softroute = rt
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
		if rt, exists = sr.hardroute[pathsplit[0]]; !exists {
			rt = NewSunnyRoute()
			sr.hardroute[pathsplit[0]] = rt
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
		trts[0] = sr
		copy(trts[1:len(trts)], rts)
		rts = trts

		tvarnames := make([][]string, len(varnames)+1)
		tvarnames[0] = []string{varname}
		copy(tvarnames[1:len(tvarnames)], varnames)
		varnames = tvarnames
	}

	return
}

func (sr *SunnyRoute) HasRoute(p string) bool {
	ep, _, _ := sr.FindEndPoint(sr.SplitPath(p), make([]string, 0, 3))
	return ep != nil
}

func (sr *SunnyRoute) SplitPath(p string) (pathsplit []string) {
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

func (sr *SunnyRoute) Handle(p string, h interface{}, method ...string) (ep EndPoint) {
	p = strings.TrimSpace(p)

	if p == "" {
		if sr.hardend != nil {
			ep = sr.hardend
		} else {
			ep = &SunnyEndPoint{}
			sr.hardend = ep
		}
		ep.SetHandler(h, method...)
	} else if p == "/" {
		if sr.softend != nil {
			ep = sr.softend
		} else {
			ep = &SunnyEndPoint{}
			sr.softend = ep
		}
		ep.SetHandler(h, method...)
	} else {
		varnames, rts := sr.BuildRoute(p)

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

func (sr *SunnyRoute) FindEndPoint(p []string, data []string) (EndPoint, []string, []string) {
	var lpath = len(p)

	if data == nil {
		data = make([]string, 0, 3)
	}

	if lpath <= 0 {
		if sr.hardend != nil {
			return sr.hardend, p, data
		} else {
			return sr.softend, p, data
		}
	}

	var curpath = p[0]
	var noext = ""
	var fullp = p
	p = p[1:lpath]

	if lpath == 1 && path.Ext(curpath) != "" {
		noext = strings.TrimSuffix(curpath, path.Ext(curpath))
	}

	if route, exists := sr.hardroute[curpath]; exists {
		return route.FindEndPoint(p, data)
	} else if noext != "" {
		if route, exists := sr.hardroute[curpath]; exists {
			return route.FindEndPoint(p, data)
		}
	}

	for regx, route := range sr.regxroute {
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

	for rtype, route := range sr.typeroute {
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

	if sr.softroute != nil {
		return nextEndPoint(sr.softroute, typepath, p, data)
	}

	p = fullp
	return sr.softend, p, data
}

func (sr *SunnyRoute) FindRequestedEndPoint(p string, r *http.Request) *RequestedEndPoint {
	pathsplit := sr.SplitPath(p)
	ep, upath, data := sr.FindEndPoint(pathsplit, make([]string, 0, 3))
	if ep != nil {
		return ep.GetRequestedEndPoint(r, upath, data)
	}
	return nil
}

func (sr *SunnyRoute) HardEndPoint() EndPoint {
	return sr.hardend
}

func (sr *SunnyRoute) SoftEndPoint() EndPoint {
	return sr.softend
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

func (se *SunnyEndPoint) PrependVarName(names ...string) {
	var lnames = len(names)
	if lnames == 0 {
		return
	}

	var varnames = make([]string, len(se.varnames)+lnames)

	copy(varnames[0:lnames], names)

	if lvnames := len(varnames); lvnames > lnames {
		copy(varnames[lnames:lvnames], se.varnames)
	}

	se.varnames = varnames
}

func (se *SunnyEndPoint) AppendVarName(name ...string) {
	se.varnames = append(se.varnames, name...)
}

func (se *SunnyEndPoint) SetHandler(handler interface{}, method ...string) error {
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
		se.get = h
		se.post = h
		se.put = h
		se.patch = h
		se.delete = h
	} else {
		for _, m := range method {
			switch strings.ToUpper(m) {
			case "GET":
				se.get = h
			case "POST":
				se.post = h
			case "PUT":
				se.put = h
			case "PATCH":
				se.patch = h
			case "DELETE":
				se.delete = h
			case "HEAD":
				se.head = h
			}
		}
	}

	return nil
}

func (se *SunnyEndPoint) Handlers() map[string]http.Handler {
	return map[string]http.Handler{
		"GET":    se.get,
		"POST":   se.post,
		"PUT":    se.put,
		"PATCH":  se.patch,
		"DELETE": se.delete,
		"HEAD":   se.HeadHandler(),
	}
}

func (se *SunnyEndPoint) GetHandler() http.Handler {
	return se.get
}

func (se *SunnyEndPoint) PostHandler() http.Handler {
	return se.post
}

func (se *SunnyEndPoint) PutHandler() http.Handler {
	return se.put
}

func (se *SunnyEndPoint) PatchHandler() http.Handler {
	return se.patch
}

func (se *SunnyEndPoint) DeleteHandler() http.Handler {
	return se.delete
}

func (se *SunnyEndPoint) HeadHandler() (head http.Handler) {
	if head = se.head; head == nil && se.get != nil {
		head = se.get
	}
	return
}

func (se *SunnyEndPoint) Handler(method string) http.Handler {
	method = strings.ToUpper(method)
	return se.Handlers()[method]
}

func (se *SunnyEndPoint) GetRequestedEndPoint(r *http.Request, upath []string, data []string) *RequestedEndPoint {
	handlers := se.Handlers()
	if h, exists := handlers[r.Method]; exists && h != nil {
		var pdata = make(map[string]string)
		for i, value := range data {
			if se.varnames[i] != "_" {
				pdata[se.varnames[i]] = value
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
			EndPoint: se,
		}
	}

	return nil
}

func (se *SunnyEndPoint) Methods() (methods []string) {
	methods = make([]string, 0, 6)
	handlers := se.Handlers()

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
