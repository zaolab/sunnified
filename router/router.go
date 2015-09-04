package router

import (
	//"github.com/zaolab/sunnified/config"
	"github.com/zaolab/sunnified/web"
	"net/http"
	"regexp"
	"strings"
)

func NewSunnyRouter() *SunnyRouter {
	return &SunnyRouter{
		Route:     NewSunnyRoute(),
		hostregex: make([]*regexp.Regexp, 0, 5),
		hostsplit: make([]string, 0, 5),
		routers:   make(map[string]Router),
		matchers:  make(map[string]RouteMatcher),
	}
}

type OptionsHandler interface {
	ServeOptions(http.ResponseWriter, *http.Request, map[string]string)
}

type EndPointOrigin interface {
	Origin() map[string]string
}

type Router interface {
	CanRouteRequest(*http.Request, map[string]interface{}) (bool, map[string]interface{})
	SetMatcher(name string, rm RouteMatcher)
	DeleteMatcher(name string)
	Matcher(name string) RouteMatcher
	SubRouter(string) Router
	AddRouter(string, Router) bool
	DelRouter(Router)
	DelRouterByName(string)
	SetParent(Router) bool
	ResetParent()
	Parent() Router
	HasParent() bool
	FindRequestedEndPoint(value map[string]interface{}, r *http.Request) (Router, *RequestedEndPoint)
	ServeHTTP(http.ResponseWriter, *http.Request)
	ServeRequestedEndPoint(http.ResponseWriter, *http.Request, *RequestedEndPoint)
	Handle(string, interface{}, ...string) EndPoint
}

type RouterPathPrefix interface {
	SetPathPrefix(path string, canon string)
	PathPrefix() string
	FullPathPrefix() string // the full path prefix including its parent's & ancestors'
	PathPrefixCanon() string
	FullPathPrefixCanon() string
}

type RouterHost interface {
	SetHost(host string, canon string)
	Host() string
	FullHost() string
	HostCanon() string
	FullHostCanon() string
}

type EndPoint interface {
	Handlers() map[string]http.Handler
	GetHandler() http.Handler
	PostHandler() http.Handler
	PutHandler() http.Handler
	PatchHandler() http.Handler
	DeleteHandler() http.Handler
	HeadHandler() http.Handler
	SetHandler(interface{}, ...string) error
	Methods() []string
	GetRequestedEndPoint(*http.Request, []string, []string) *RequestedEndPoint
	PrependVarName(...string)
	AppendVarName(...string)
}

type Route interface {
	HasRoute(string) bool
	FindRequestedEndPoint(p string, r *http.Request) *RequestedEndPoint
	FindEndPoint(p []string, data []string) (EndPoint, []string, []string)
	HardEndPoint() EndPoint
	SoftEndPoint() EndPoint
	Switch(string) Switch
	BuildRoute(string) ([][]string, []Route)
	Handle(string, interface{}, ...string) EndPoint
}

type Switch interface {
	HasRoute(string) bool
	Handle(string, http.Handler) EndPoint
	BuildRoute(string) ([][]string, []Route)
	Switch(string) Switch
	PrependVarName(...string)
}

type RouteMatcher interface {
	Match(r *http.Request, value map[string]interface{}) (bool, map[string]interface{})
}

type RouteMatcherFunc func(r *http.Request, value map[string]interface{}) (bool, map[string]interface{})

func (f RouteMatcherFunc) Match(r *http.Request, value map[string]interface{}) (bool, map[string]interface{}) {
	return f(r, value)
}

type RequestedEndPoint struct {
	Ext      string
	UPath    web.UPath
	PData    web.PData
	Method   string
	Handler  http.Handler
	EndPoint EndPoint
}

type SunnyRouter struct {
	Route

	host       string
	hostcanon  string
	hostregex  []*regexp.Regexp
	hostsplit  []string
	hostpredot bool
	pathprefix string
	pathcanon  string

	parent   Router
	routers  map[string]Router
	matchers map[string]RouteMatcher
}

func (this *SunnyRouter) CanRouteRequest(r *http.Request, value map[string]interface{}) (bool, map[string]interface{}) {
	var ok bool

	if value == nil {
		value = make(map[string]interface{})
	}

	for _, matcher := range this.matchers {
		if ok, value = matcher.Match(r, value); ok {
			return false, value
		}
	}

	return true, value
}

func (this *SunnyRouter) SetHost(host string, canon string) {
	this.host = strings.ToLower(host)
	this.hostcanon = canon
	this.hostpredot = false
	this.hostsplit = nil
	this.hostregex = nil

	if host != "" {
		if host[0] == '.' {
			host = host[1:len(host)]
			this.hostpredot = true
		}

		this.hostsplit = strings.Split(host, ".")
		this.hostregex = make([]*regexp.Regexp, len(this.hostsplit))

		for i, v := range this.hostsplit {
			if strings.ContainsAny(v, "?|()*[]") {
				this.hostregex[i] = regexp.MustCompile("^" + strings.Replace(v, "*", ".*", -1) + "$")
			}
		}

		if this.matchers == nil {
			this.matchers = make(map[string]RouteMatcher)
		}

		this.matchers["host"] = RouteMatcherFunc(this.MatchHost)
	} else {
		delete(this.matchers, "host")
	}
}

func (this *SunnyRouter) Host() string {
	return this.host
}

func (this *SunnyRouter) FullHost() string {
	var getParentHost func(Router) string

	getParentHost = func(rt Router) string {
		p := rt.Parent()
		if p != nil {
			if pp, ok := p.(RouterHost); ok {
				return pp.FullHost()
			} else {
				return getParentHost(p)
			}
		} else {
			return ""
		}
	}

	if h := getParentHost(this); h != "" {
		if h[0] == '.' {
			return this.host + h
		}

		return this.host + "." + h
	}

	return this.host
}

func (this *SunnyRouter) HostCanon() string {
	return this.hostcanon
}

func (this *SunnyRouter) FullHostCanon() string {
	var getParentHost func(Router) string

	getParentHost = func(rt Router) string {
		p := rt.Parent()
		if p != nil {
			if pp, ok := p.(RouterHost); ok {
				return pp.FullHostCanon()
			} else {
				return getParentHost(p)
			}
		} else {
			return ""
		}
	}

	if h := getParentHost(this); h != "" {
		return this.hostcanon + "." + h
	}

	return this.hostcanon
}

func (this *SunnyRouter) MatchHost(r *http.Request, value map[string]interface{}) (bool, map[string]interface{}) {
	if vIface, exists := value["host"]; exists {
		v := vIface.(string)

		if this.pathprefix == "" {
			return true, value
		}

		var (
			hostArr  = strings.Split(v, ".")
			lohArr   = len(this.hostsplit)
			lhostArr = len(hostArr)
		)

		if lhostArr < lohArr || (lhostArr > lohArr && !this.hostpredot) {
			return false, value
		} else if lhostArr > lohArr {
			hostArr = hostArr[lhostArr-lohArr : lhostArr]
		}

		for i, re := range this.hostregex {
			if re != nil {
				if !re.MatchString(hostArr[i]) {
					return false, value
				}
			} else if this.hostsplit[i] != hostArr[i] {
				return false, value
			}
		}

		value["host"] = strings.Join(hostArr[0:lhostArr-lohArr], ".")
		return true, value
	}

	host := strings.ToLower(r.Host)
	h := this.FullHost()

	if h == "" {
		return true, value
	}

	if strings.Contains(host, ":") {
		host = host[0:strings.Index(host, ":")]
	}

	if strings.ContainsAny(h, "?|()*[]") {
		if this.hostpredot {
			h = h[1:len(h)]
		}

		var (
			hArr     = strings.Split(h, ".")
			hostArr  = strings.Split(host, ".")
			_hostArr = hostArr
			lhArr    = len(hArr)
			lohArr   = len(this.hostsplit)
			lhostArr = len(hostArr)
		)

		if lhostArr < lhArr || (lhostArr > lhArr && !this.hostpredot) {
			return false, value
		} else if lhostArr > lhArr {
			hostArr = hostArr[lhostArr-lhArr : lhostArr]
		}

		for i, v := range hostArr {
			if i >= lohArr {
				j := i - lohArr
				if strings.ContainsAny(hArr[j], "?|()*[]") {
					tmp := "^" + strings.Replace(hArr[j], "*", ".*", -1) + "$"
					if ok, err := regexp.MatchString(tmp, v); !ok || err != nil {
						return false, value
					}
				} else if hArr[j] != v {
					return false, value
				}
			} else {
				if this.hostregex[i] != nil {
					if !this.hostregex[i].MatchString(v) {
						return false, value
					}
				} else if this.hostsplit[i] != v {
					return false, value
				}
			}
		}

		value["host"] = strings.Join(_hostArr[0:lhostArr-lhArr], ".")
		return true, value
	}

	/* if starts with '.' char, we will match all child host names
	 * e.g.
	 * .abc.com
	 * will match
	 * a.abc.com, a.a.abc.com, a.a.a.abc.com, b.abc.com, etc.
	 * but it does not match
	 * babc.com */
	if h != "" && h[0] == '.' {
		lenh := len(h)
		lenhost := len(host)
		h = h[1:lenh]
		lenh -= 1

		if lenhost < lenh || (lenhost > lenh && host[lenhost-lenh-1] != '.') {
			return false, value
		}

		if host[lenhost-lenh:lenhost] == h {
			value["host"] = host[0 : lenhost-lenh]
			return true, value
		} else {
			return false, value
		}
	}

	return host == h, value
}

func (this *SunnyRouter) SetPathPrefix(path string, canon string) {
	path = strings.TrimSuffix(path, "/")

	if path != "" && path[0] != '/' {
		path = "/" + path
	}
	if canon != "" && canon[0] != '/' {
		canon = "/" + canon
	}

	this.pathprefix = path
	this.pathcanon = canon

	if path != "" {
		if this.matchers == nil {
			this.matchers = make(map[string]RouteMatcher)
		}

		this.matchers["pathprefix"] = RouteMatcherFunc(this.MatchPathPrefix)
	} else {
		delete(this.matchers, "pathprefix")
	}
}

func (this *SunnyRouter) MatchPathPrefix(r *http.Request, value map[string]interface{}) (bool, map[string]interface{}) {
	if vIface, exists := value["pathprefix"]; exists {
		v := vIface.(string)

		lenpp := len(this.pathprefix)
		lenv := len(v)
		if this.pathprefix == "" || (lenv > lenpp && v[0:lenpp] == this.pathprefix) {
			value["pathprefix"] = v[lenpp:lenv]
			return true, value
		} else {
			return false, value
		}
	}

	var (
		path    = this.FullPathPrefix()
		urlpath = r.URL.Path
		lpath   = len(path)
	)

	if path == "" {
		return true, value
	}

	if urlpath != "" && urlpath[len(urlpath)-1] != '/' {
		urlpath += "/"
	}

	return len(urlpath) >= lpath && urlpath[0:lpath] == path, nil
}

func (this *SunnyRouter) PathPrefix() string {
	return this.pathprefix
}

func (this *SunnyRouter) FullPathPrefix() string {
	var getParentPath func(Router) string

	getParentPath = func(rt Router) string {
		p := rt.Parent()
		if p != nil {
			if pp, ok := p.(RouterPathPrefix); ok {
				return pp.FullPathPrefix()
			} else {
				return getParentPath(p)
			}
		} else {
			return ""
		}
	}

	if prefix := getParentPath(this); prefix != "" {
		return prefix + this.pathprefix
	}

	return this.pathprefix
}

func (this *SunnyRouter) PathPrefixCanon() string {
	return this.pathcanon
}

func (this *SunnyRouter) FullPathPrefixCanon() string {
	var getParentPath func(Router) string

	getParentPath = func(rt Router) string {
		p := rt.Parent()
		if p != nil {
			if pp, ok := p.(RouterPathPrefix); ok {
				return pp.FullPathPrefixCanon()
			} else {
				return getParentPath(p)
			}
		} else {
			return ""
		}
	}

	if prefix := getParentPath(this); prefix != "" {
		prefix = strings.TrimSuffix(prefix, "/")
		return prefix + this.pathcanon
	}

	return this.pathcanon
}

func (this *SunnyRouter) SetMatcher(name string, rm RouteMatcher) {
	if this.matchers == nil {
		this.matchers = make(map[string]RouteMatcher)
	}

	this.matchers[name] = rm
}

func (this *SunnyRouter) DeleteMatcher(name string) {
	delete(this.matchers, name)
}

func (this *SunnyRouter) Matcher(name string) RouteMatcher {
	if m, ok := this.matchers[name]; ok {
		return m
	}
	return nil
}

func (this *SunnyRouter) SubRouter(name string) (rt Router) {
	rt = NewSunnyRouter()
	this.AddRouter(name, rt)
	return
}

func (this *SunnyRouter) AddRouter(name string, rt Router) (ok bool) {
	if this.routers == nil {
		this.routers = make(map[string]Router)
	}

	if ok = rt.SetParent(this); ok {
		this.routers[name] = rt
	}

	return
}

func (this *SunnyRouter) DelRouterByName(name string) {
	if rt, exists := this.routers[name]; exists {
		delete(this.routers, name)
		rt.ResetParent()
	}
}

func (this *SunnyRouter) DelRouter(rt Router) {
	var rtname = ""

	for name, r := range this.routers {
		if r == rt {
			rtname = name
			break
		}
	}

	if rtname != "" {
		delete(this.routers, rtname)
		rt.ResetParent()
	}
}

func (this *SunnyRouter) Routers() map[string]Router {
	newcopy := make(map[string]Router)
	for k, v := range this.routers {
		newcopy[k] = v
	}
	return newcopy
}

func (this *SunnyRouter) Router(name string) Router {
	if rt, exists := this.routers[name]; exists {
		return rt
	}

	return nil
}

func (this *SunnyRouter) SetParent(rt Router) bool {
	if this.parent == nil {
		this.parent = rt
		return true
	}
	return false
}

func (this *SunnyRouter) ResetParent() {
	parent := this.parent
	this.parent = nil

	if parent != nil {
		parent.DelRouter(this)
	}
}

func (this *SunnyRouter) Parent() Router {
	return this.parent
}

func (this *SunnyRouter) HasParent() bool {
	return this.parent != nil
}

func (this *SunnyRouter) FindRequestedEndPoint(value map[string]interface{}, r *http.Request) (Router, *RequestedEndPoint) {
	var ok bool

	if value == nil {
		value = map[string]interface{}{
			"pathprefix": r.URL.Path,
		}
	} else if _, exists := value["pathprefix"]; !exists {
		value["pathprefix"] = r.URL.Path
	}

	if ok, value = this.CanRouteRequest(r, value); ok {
		for _, rt := range this.routers {
			if rt, rep := rt.FindRequestedEndPoint(value, r); rep != nil {
				return rt, rep
			}
		}

		return this, this.Route.FindRequestedEndPoint(value["pathprefix"].(string), r)
	}

	return this, nil
}

func (this *SunnyRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router, rep := this.FindRequestedEndPoint(make(map[string]interface{}), r)
	router.ServeRequestedEndPoint(w, r, rep)
}

func (this *SunnyRouter) ServeRequestedEndPoint(w http.ResponseWriter, r *http.Request, rep *RequestedEndPoint) {
	if rep != nil {
		ctxt := web.NewContext(w, r)
		ctxt.UPath = rep.UPath
		ctxt.PData = rep.PData

		if !HandleHeaders(ctxt, rep.EndPoint, rep.Handler) {
			if ctxthandler, ok := rep.Handler.(web.ContextHandler); ok {
				ctxthandler.ServeContextHTTP(ctxt)
			} else {
				rep.Handler.ServeHTTP(w, r)
			}
		}
	} else {
		http.NotFound(w, r)
	}
}

func HandleHeaders(ctxt *web.Context, ep EndPoint, h http.Handler) (served bool) {
	var origin map[string]string

	if originget, ok := ep.(EndPointOrigin); ok {
		origin = originget.Origin()
	}

	if ctxt.Request.Method == "OPTIONS" {
		if opthandler, ok := h.(web.ContextOptionsHandler); ok {
			opthandler.ServeContextOptions(ctxt, origin)
		}
		if opthandler, ok := h.(OptionsHandler); ok {
			opthandler.ServeOptions(ctxt.Response, ctxt.Request, origin)
		} else {
			ServeOptions(ep.Methods(), ctxt.Response, ctxt.Request, origin)
		}

		served = true
	} else {
		if originhead := ctxt.Request.Header.Get("Origin"); originhead != "" {
			SetHeaderOrigin(ctxt.Response, ctxt.Request, origin)
		}
	}

	return
}

func SetHeaderOrigin(w http.ResponseWriter, r *http.Request, origin map[string]string) {
	rheader := r.Header
	originhead := rheader.Get("Origin")

	if originhead == "" {
		return
	}

	header := w.Header()
	allow := false

	if len(origin) > 0 {
		if alloworigin, ok := origin["Access-Control-Allow-Origin"]; ok {
			var originlist []string

			if strings.Contains(alloworigin, ",") {
				originlist = strings.Split(alloworigin, ",")
			} else if strings.Contains(alloworigin, " ") {
				originlist = strings.Split(alloworigin, " ")
			} else {
				originlist = []string{alloworigin}
			}

			for _, o := range originlist {
				o = strings.TrimSpace(o)

				if strings.ToLower(o) == strings.ToLower(originhead) {
					header.Set("Access-Control-Allow-Origin", originhead)
					header.Add("Vary", "Origin")
					allow = true
					break
				} else if o == "*" {
					header.Set("Access-Control-Allow-Origin", "*")
					allow = true
					break
				}
			}

			if allow {
				for hkey, hval := range origin {
					if hkey == "Access-Control-Allow-Origin" {
						continue
					}
					header.Set(hkey, hval)
				}

				if reqheader := rheader.Get("Access-Control-Request-Headers"); reqheader != "" {
					if header.Get("Access-Control-Allow-Headers") == "*" {
						header.Set("Access-Control-Allow-Headers", reqheader)
					}
				} else {
					header.Del("Access-Control-Allow-Headers")
				}

				if reqmethod := rheader.Get("Access-Control-Request-Method"); reqmethod != "" {
					// by default let's allow all methods
					if header.Get("Access-Control-Allow-Methods") == "" {
						header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
					}
				} else {
					header.Del("Access-Control-Allow-Methods")
				}
			}
		}
	}

	if !allow {
		header.Del("Access-Control-Allow-Origin")
	}
}

func ServeOptions(methods []string, w http.ResponseWriter, r *http.Request, origin map[string]string) {
	header := w.Header()
	methstr := "HEAD, OPTIONS, GET, POST, PUT, PATCH, DELETE"

	if methods != nil {
		methstr = strings.Join(methods, ", ")
		if strings.Contains(methstr, "GET") && !strings.Contains(methstr, "HEAD") {
			methstr += ", HEAD"
		}
		// we are already serving OPTIONS here...
		if !strings.Contains(methstr, "OPTIONS") {
			methstr += ", OPTIONS"
		}
	}

	header.Set("Allow", methstr)
	SetHeaderOrigin(w, r, origin)
	w.WriteHeader(200)
}
