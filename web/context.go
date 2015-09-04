package web

import (
	"bytes"
	"errors"
	//"github.com/zaolab/sunnified/router"
	"github.com/zaolab/sunnified/util"
	"github.com/zaolab/sunnified/util/collection"
	"github.com/zaolab/sunnified/util/event"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrResourceNotFound = errors.New("Resource not found")

const REQMETHOD_X_METHOD_NAME = "X-HTTP-Method-Override"

const (
	USER_ANONYMOUS int = iota
	USER_USER
	USER_PREMIUMUSER
	USER_WRITER
	USER_SUPERWRITER
	USER_MODERATOR
	USER_SUPERMODERATOR
	USER_ADMIN
	USER_SUPERADMIN
)

type Q map[string]string

type SessionManager interface {
	ID() string
	String(string) string
	Int(string) int
	Int64(string) int64
	Float32(string) float32
	Float64(string) float64
	Bool(string) bool
	Byte(string) byte
	Get(string) interface{}
	MapValue(string, interface{})
	IPAddress() string
	UserAgent() string
	Created() time.Time
	Accessed() time.Time
	Expiry() time.Time
	AuthUser() UserModel
	Set(string, interface{})
	Remove(string)
	SetIPAddress(string)
	SetUserAgent(string)
	SetExpiry(time.Time)
	SetAuthUser(UserModel)
	SetAuthUserData(id string, email string, name string, lvl int)
	SetAnonymous()
	IsAuthUser(id string) bool
	UpdateAccessed()
	AddFlash(string)
	HasFlash() bool
	Flash() string
	AllFlashes() []string
	PeekFlashes() []string
	LenFlashes() int
}

type UserModel interface {
	ID() string
	Email() string
	Name() string
	Level() int
	IsSuperAdmin() bool
	IsAdmin() bool
	IsSuperModerator() bool
	IsModerator() bool
	IsSuperWriter() bool
	IsWriter() bool
	IsPremiumUser() bool
	IsUser() bool
	IsAnonymous() bool
}

type CacheManager interface {
	Get(string) interface{}
	MapValue(string, interface{})
	Set(string, interface{}, time.Duration)
	Delete(string)
	Clear()
}

type RedirectError struct {
	code int
	url  string
}

type ContextHandler interface {
	ServeContextHTTP(*Context)
}

type ContextOptionsHandler interface {
	ServeContextOptions(*Context, map[string]string)
}

func (this RedirectError) Error() string {
	return "Redirecting service to " + this.url + "."
}

func (this RedirectError) URL() string {
	return this.url
}

func (this RedirectError) Code() int {
	return this.code
}

type AppError struct {
	code int
	err  string
}

func (this AppError) Error() string {
	return this.err
}

func (this AppError) Code() int {
	return this.code
}

func XMethod(r *http.Request) (xmeth string) {
	xmeth = r.Method

	if r.Method == "POST" {
		tmpxmeth := r.Header.Get(REQMETHOD_X_METHOD_NAME)

		if tmpxmeth == "" {
			tmpxmeth = r.PostFormValue(REQMETHOD_X_METHOD_NAME)
		}

		switch tmpxmeth {
		case "GET":
			fallthrough
		case "POST":
			fallthrough
		case "PUT":
			fallthrough
		case "PATCH":
			fallthrough
		case "DELETE":
			xmeth = tmpxmeth
		}
	}

	return
}

type Context struct {
	mutex       sync.RWMutex
	SetTitle_   func(string)
	Request     *http.Request
	Response    http.ResponseWriter
	UPath       UPath
	PData       PData
	Module      string
	Controller  string
	Action      string
	Ext         string
	Event       *event.EventRouter
	Session     SessionManager
	Cache       CacheManager
	resource    map[string]interface{}
	sunnyserver int
	issunny     bool
	time        time.Time
	redirecting bool
	errorcode   int
	err         string
	flashcache  *collection.Queue
	//Router      *router.Router // TODO: change it into an interface so there is no cyclic reference when router uses web.UPath, web.FDATA
}

func (this *Context) AppError(err string, status ...int) {
	this.SetError(err, status...)
	panic(AppError{code: this.errorcode, err: err})
}

func (this *Context) SetErrorCode(status int) {
	this.errorcode = status
}

func (this *Context) ErrorCode() int {
	return this.errorcode
}

func (this *Context) SetError(err string, status ...int) {
	if len(status) > 0 {
		this.errorcode = status[0]
	} else {
		this.errorcode = 500
	}

	this.err = err
}

func (this *Context) HasErrorCode() bool {
	return this.HasError()
}

func (this *Context) HasError() bool {
	return this.errorcode != 200 && this.errorcode != 0
}

func (this *Context) Error() string {
	return this.err
}

func (this *Context) SetTitle(title string) {
	if this.SetTitle_ != nil {
		this.SetTitle_(title)
	}
}

func (this *Context) ReqHeader(header string) string {
	return this.Request.Header.Get(header)
}

func (this *Context) ReqHeaderHas(header string, value string) bool {
	return strings.Contains(this.Request.Header.Get(header), value)
}

func (this *Context) ReqHeaderIs(header string, value string) bool {
	return this.Request.Header.Get(header) == value
}

func (this *Context) ResHeader(header string) string {
	return this.Response.Header().Get(header)
}

func (this *Context) SetHeader(header string, value string) {
	this.Response.Header().Set(header, value)
}

func (this *Context) AddHeader(header string, value string) {
	this.Response.Header().Add(header, value)
}

func (this *Context) StartTime() time.Time {
	return this.time
}

func (this *Context) SunnyServerId() int {
	// to prevent mistaking of sunny server when Context created without
	// using NewContext
	if this.issunny {
		return this.sunnyserver
	} else {
		return -1
	}
}

func (this *Context) IsSunnyContext() bool {
	return this.issunny
}

func (this *Context) IsRedirecting() bool {
	return this.redirecting
}

func (this *Context) IsHttps() bool {
	return this.Request.TLS != nil
}

func (this *Context) IsCors() bool {
	return this.Request.Header.Get("Origin") != ""
}

func (this *Context) IsAjax() bool {
	return strings.ToLower(this.Request.Header.Get("X-Requested-With")) == "xmlhttprequest"
}

func (this *Context) IsAjaxOrCors() bool {
	return this.IsAjax() || this.IsCors()
}

func (this *Context) FwdedForOrRmteAddr() (ip net.IP) {
	if ipstr := this.Request.Header.Get("X-Forwarded-For"); ipstr == "" {
		ip = this.RemoteAddress()
	} else if cindex := strings.Index(ipstr, ","); cindex != -1 {
		ip = net.ParseIP(ipstr[0:cindex])
	} else {
		ip = net.ParseIP(ipstr)
	}
	return
}

func (this *Context) RemoteAddress() net.IP {
	raddr := this.Request.RemoteAddr
	if index := strings.Index(raddr, ":"); index != -1 {
		raddr = raddr[0:index]
	}

	return net.ParseIP(raddr)
}

func (this *Context) XRealIPOrRmteAddr() (ip net.IP) {
	if ipstr := this.Request.Header.Get("X-Real-IP"); ipstr == "" {
		ip = this.RemoteAddress()
	} else {
		ip = net.ParseIP(ipstr)
	}
	return
}

func (this *Context) RequestValue(name string) string {
	return this.Request.FormValue(name)
}

func (this *Context) RequestValues(name string) []string {
	return this.Request.Form[name]
}

func (this *Context) PostValue(name string) string {
	return this.Request.PostFormValue(name)
}

func (this *Context) PostValues(name string) []string {
	return this.Request.PostForm[name]
}

func (this *Context) Method() string {
	return this.Request.Method
}

func (this *Context) XMethod() string {
	return XMethod(this.Request)
}

func (this *Context) SetETag(etag string) {
	this.Response.Header().Set("ETag", etag)
}

func (this *Context) SetCookie(c *http.Cookie) {
	if this.IsHttps() && !c.Secure {
		c.Secure = true
	}
	http.SetCookie(this.Response, c)
}

func (this *Context) SetCookieValue(name string, value string) {
	this.SetCookie(&http.Cookie{
		Name:  name,
		Value: value,
	})
}

func (this *Context) DeleteCookie(cname string) {
	this.SetCookie(&http.Cookie{
		Name:   cname,
		MaxAge: -1,
	})
}

func (this *Context) Cookie(cname string) (c *http.Cookie) {
	c, _ = this.Request.Cookie(cname)
	return
}

func (this *Context) CookieValue(cname string) string {
	c := this.Cookie(cname)
	if c != nil {
		return c.Value
	}
	return ""
}

func (this *Context) AddFlash(msg string) {
	if this.Session != nil {
		this.Session.AddFlash(msg)
	} else {
		if this.flashcache == nil {
			this.flashcache = collection.NewQueue(msg)
		} else {
			this.flashcache.Push(msg)
		}
	}
}

func (this *Context) HasFlash() bool {
	if this.Session != nil {
		return this.Session.HasFlash()
	} else {
		return this.flashcache != nil && this.flashcache.HasQueue()
	}
}

func (this *Context) Flash() string {
	if this.Session != nil {
		return this.Session.Flash()
	} else if this.flashcache != nil {
		return this.flashcache.PullDefault("").(string)
	}
	return ""
}

func (this *Context) AllFlashes() []string {
	if this.Session != nil {
		return this.Session.AllFlashes()
	} else if this.flashcache != nil {
		flashes := this.PeekFlashes()
		this.flashcache.Clear()
		return flashes
	}

	return nil
}

func (this *Context) PeekFlashes() []string {
	if this.Session != nil {
		return this.Session.PeekFlashes()
	} else if this.flashcache != nil {
		var flash string
		flashes := make([]string, 0, this.flashcache.Len())

		for iter := this.flashcache.Iterator(); iter.Next(&flash); {
			flashes = append(flashes, flash)
		}

		return flashes
	}

	return nil
}

func (this *Context) LenFlashes() int {
	if this.Session != nil {
		return this.Session.LenFlashes()
	} else if this.flashcache != nil {
		return this.flashcache.Len()
	}
	return 0
}

func (this *Context) PrivateNoStore() {
	header := this.Response.Header()
	header.Set("Pragma", "no-cache")
	header.Set("Cache-Control", "private, max-age=0, no-cache, no-store")
}

func (this *Context) PrivateNoCache() {
	header := this.Response.Header()
	header.Set("Pragma", "no-cache")
	header.Set("Cache-Control", "private, max-age=0, no-cache")
}

func (this *Context) PublicCache(age int) {
	header := this.Response.Header()

	if pragma, exists := header["Pragma"]; exists {
		for i, p := range pragma {
			if p == "no-cache" {
				if lenpragma := len(pragma); lenpragma > 1 {
					newslice := make([]string, lenpragma-1)
					copy(newslice, pragma[0:i])
					if lenpragma > i+1 {
						copy(newslice[i:], pragma[i+1:])
					}
					header["Pragma"] = newslice
					break
				} else {
					delete(header, "Pragma")
				}
			}
		}
	}

	if age <= 0 {
		age = 31536000 // a year
	}

	header.Set("Cache-Control", "public, max-age="+strconv.Itoa(age))
}

// TODO: make this redirect only within internal urls and make a RedirectOut to redirect to any url
func (this *Context) SetRedirect(location string, state ...int) (status int) {
	if !this.redirecting {
		if !strings.HasPrefix(location, "http") {
			location = this.URL(location)
		}

		status = 303

		if len(state) > 0 {
			status = state[0]
		}

		http.Redirect(this.Response, this.Request, location, status)
		this.redirecting = true
	}

	return
}

func (this *Context) Redirect(location string, state ...int) {
	if !this.redirecting {
		panic(RedirectError{code: this.SetRedirect(location, state...), url: location})
	}
}

func (this *Context) URL(path string, qstr ...Q) string {
	var buf bytes.Buffer

	if !strings.HasPrefix(path, "http") {
		buf.WriteString("http")
		if this.Request.TLS != nil {
			buf.WriteByte('s')
		}
		buf.WriteString("://")
	}

	hasstar := strings.HasPrefix(path, "*")

	if host, pathl := strings.ToLower(this.Request.Host), strings.ToLower(path); !strings.Contains(pathl, host) {
		buf.WriteString(host)
		if !hasstar && pathl != "" && pathl[0] != '/' {
			buf.WriteString("/")
		}
	}

	if hasstar {
		// TODO: switch .URL.Path to .RequestURI instead
		upath := this.Request.URL.Path
		if upath != "" && upath[len(upath)-1] == '/' {
			upath = upath[0 : len(upath)-1]
		}
		buf.WriteString(upath)
		buf.WriteString(path[1:])
	} else {
		buf.WriteString(path)
	}

	if len(qstr) > 0 && len(qstr[0]) > 0 {
		buf.WriteByte('?')

		for i := range qstr {
			for k, v := range qstr[i] {
				buf.WriteString(url.QueryEscape(k))
				buf.WriteByte('=')
				buf.WriteString(url.QueryEscape(v))
				buf.WriteByte('&')
			}
		}

		buf.Truncate(buf.Len() - 1)
	}

	return buf.String()
}

func (this *Context) QueryStr(s ...string) (qs Q) {
	qs = make(Q)
	isname := true
	name := ""

	for _, v := range s {
		if isname {
			name = v
			isname = false
		} else {
			qs[name] = v
		}
	}
	return
}

func (this *Context) MapResourceValue(name string, ref interface{}) (err error) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	if val, ok := this.resource[name]; ok && ref != nil {
		return util.MapValue(ref, val)
	}

	return ErrResourceNotFound
}

func (this *Context) Resource(name string) (val interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	val, _ = this.resource[name]

	return
}

func (this *Context) SetResource(name string, ref interface{}) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.resource[name] = ref
}

/*
func (this *Context) SaveFile(dst string, maxtime time.Duration, arrayfile bool, inputname ...string) error {
	rdr, err := this.Request.MultipartReader()
	if err != nil {
		return err
	}

	ch := time.After(maxtime)

	for {
		part, err := rdr.NextPart()
		if err == io.EOF {
			break
		}
		if !In(part.FormName(), inputname) {
			continue
		}
		file, err := ioutil.TempFile("", "sunnycontext-")
		if err != nil {
			return err
		}
		defer file.Close()
		io.LimitReader{part,2048}
	}
}

func In(s string, arr []string) bool {
	for ts := range arr {
		if s == ts {
			return true
		}
	}
	return false
}
*/

func (this *Context) Close() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.resource = nil
	this.Session = nil
	this.Cache = nil
	this.Event = nil
	this.UPath = nil
	this.PData = nil
	this.Request = nil
	this.Response = nil
	this.SetTitle_ = nil
}

func NewSunnyContext(w http.ResponseWriter, r *http.Request, sunnyserver int) (ctxt *Context) {
	ctxt = NewContext(w, r)
	ctxt.sunnyserver = sunnyserver
	ctxt.issunny = true
	return
}

func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Request:     r,
		Response:    w,
		resource:    make(map[string]interface{}),
		sunnyserver: -1,
		time:        time.Now(),
	}
}
