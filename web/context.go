package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zaolab/sunnified/util"
	"github.com/zaolab/sunnified/util/collection"
	"github.com/zaolab/sunnified/util/event"
	"github.com/zaolab/sunnified/util/validate"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/gorilla/websocket"
)

type ResponseWriterChild interface{
	ParentResponseWriter() http.ResponseWriter
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
	MaxFileSize int64
	WebSocket   *websocket.Conn
	resource    map[string]interface{}
	sunnyserver int
	issunny     bool
	time        time.Time
	redirecting Redirection
	errorcode   int
	err         string
	flashcache  *collection.Queue
	parseState  parseState
	rdata       map[string]interface{}
	//Router      *router.Router // TODO: change it into an interface so there is no cyclic reference when router uses web.UPath, web.FDATA
}

func (this *Context) WaitRequestData() error {
	if !this.parseState.Started() {
		this.ParseRequestData()
	}

	this.parseState.Status.Wait()
	return this.parseState.Error
}

func (this *Context) parseRequestData(ctype string, body io.ReadCloser) {
	var err error

	defer func() {
		if panicerr := recover(); panicerr != nil {
			log.Println(panicerr, err)
			this.parseState.End(errors.New("Form parsing exited with panic"))
		} else {
			this.parseState.End(err)
		}
	}()

	if strings.Contains(ctype, ";") {
		ctype = strings.TrimSpace(strings.SplitN(ctype, ";", 2)[0])
	}

	if parser := GetContentParser(ctype); parser != nil {
		this.rdata = parser(body)
		requestDataToUrlValues(this.Request, this.rdata)
	} else if ctype == "application/json" {
		d := json.NewDecoder(body)
		d.UseNumber()
		d.Decode(&this.rdata)
		requestDataToUrlValues(this.Request, this.rdata)
	} else {
		// 2MB in memory
		// passing back of ErrNotMultipart is only >= golang1.3
		if err = this.Request.ParseMultipartForm(2097152); err == http.ErrNotMultipart {
			err = nil
		} else if err == nil && this.Request.MultipartForm != nil {
			if this.Request.PostForm == nil {
				this.Request.PostForm = make(url.Values)
			}
			for k, v := range this.Request.MultipartForm.Value {
				this.Request.PostForm[k] = append(this.Request.PostForm[k], v...)
			}
		}
	}
}

func (this *Context) ParseRequestData() {
	if !this.parseState.Start() {
		return
	}

	var (
		r             = this.Request
		w             = this.Response
		maxsize int64 = 10485760 //10MB
	)

	if validate.IsIn(r.Method, "POST", "PUT", "PATCH") {
		ctype := strings.ToLower(r.Header.Get("Content-Type"))
		is100 := strings.ToLower(r.Header.Get("Expect")) == "100-continue"

		if !strings.HasPrefix(ctype, "multipart") {
			if is100 && r.ContentLength > maxsize {
				goto expectFailed
			}
		} else if this.MaxFileSize > 0 {
			if maxsize = this.MaxFileSize; is100 && r.ContentLength > maxsize {
				goto expectFailed
			}

			r.Body = http.MaxBytesReader(w, r.Body, maxsize)
		}

		go this.parseRequestData(ctype, r.Body)
	} else {
		var err error

		if this.Request.Form == nil {
			this.Request.Form, err = url.ParseQuery(this.Request.URL.RawQuery)
		}

		this.parseState.End(err)
	}

	return

expectFailed:
	e := ExpectationError{size: r.ContentLength, maxsize: maxsize}
	this.parseState.End(e)
	w.Header().Set("Connection", "close")
	panic(e)
}

func (this *Context) RequestBodyData(ctype string) map[string]interface{} {
	if ctype == "" || strings.HasPrefix(strings.ToLower(this.Request.Header.Get("Content-Type")), ctype) {
		return this.rdata
	}

	return nil
}

func (this *Context) RaiseAppError(err string, status ...int) {
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

func (this *Context) RecoverError() {
	this.errorcode = 0
	this.err = ""
}

func (this *Context) AppError() (ae AppError) {
	if this.HasError() {
		ae = AppError{code: this.errorcode, err: this.err}
	}

	return
}

func (this *Context) SetTitle(title string) {
	if this.SetTitle_ != nil {
		this.SetTitle_(title)
	}
}

func (this *Context) ReqHeader(header string) string {
	return this.Request.Header.Get(header)
}

func (this *Context) ReqHeaderHas(header, value string) bool {
	return strings.Contains(this.Request.Header.Get(header), value)
}

func (this *Context) ReqHeaderIs(header, value string) bool {
	return this.Request.Header.Get(header) == value
}

func (this *Context) ResHeader(header string) string {
	return this.Response.Header().Get(header)
}

func (this *Context) SetHeader(header, value string) {
	this.Response.Header().Set(header, value)
}

func (this *Context) AddHeader(header, value string) {
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
	return this.redirecting.code > 0
}

func (this *Context) IsHttps() bool {
	return this.Request.TLS != nil
}

func (this *Context) IsCors() bool {
	return this.Request.Header.Get("Origin") != ""
}

func (this *Context) IsAjax() bool {
	return strings.ToLower(this.Request.Header.Get(HTTP_X_REQUESTED_WITH)) == "xmlhttprequest"
}

func (this *Context) IsAjaxOrCors() bool {
	return this.IsAjax() || this.IsCors()
}

func (this *Context) RedirectionCode() int {
	return this.redirecting.code
}

func (this *Context) RedirectionURL() string {
	return this.redirecting.url
}

func (this *Context) Redirection() Redirection {
	return this.redirecting
}

func (this *Context) RemoteAddress() net.IP {
	raddr := this.Request.RemoteAddr
	if index := strings.Index(raddr, ":"); index != -1 {
		raddr = raddr[0:index]
	}

	return net.ParseIP(raddr)
}

func (this *Context) FwdedForOrRmteAddr() (ip net.IP) {
	if ipstr := this.Request.Header.Get(HTTP_X_FORWARDED_FOR); ipstr != "" {
		if cindex := strings.Index(ipstr, ","); cindex != -1 {
			ip = net.ParseIP(ipstr[0:cindex])
		} else {
			ip = net.ParseIP(ipstr)
		}
	}

	if ip == nil {
		ip = this.RemoteAddress()
	}
	return
}

func (this *Context) XRealIPOrRmteAddr() (ip net.IP) {
	if ipstr := this.Request.Header.Get(HTTP_X_REAL_IP); ipstr != "" {
		ip = net.ParseIP(ipstr)
	}

	if ip == nil {
		ip = this.RemoteAddress()
	}
	return
}

func (this *Context) ClientIP(pref []string) (ip net.IP) {
	if pref != nil {
		for _, h := range pref {
			if ipstr := this.Request.Header.Get(h); ipstr != "" {
				ip = net.ParseIP(ipstr)
				if ip != nil {
					return
				}
			}
		}
	}

	ip = this.RemoteAddress()
	return
}

func (this *Context) RequestValue(name string) string {
	if this.Request.Form == nil {
		this.WaitRequestData()
	}
	return this.Request.FormValue(name)
}

func (this *Context) RequestValues(name string) []string {
	if this.Request.Form == nil {
		this.WaitRequestData()
	}
	return this.Request.Form[name]
}

func (this *Context) PostValue(name string) string {
	if this.Request.PostForm == nil {
		this.WaitRequestData()
	}
	return this.Request.PostFormValue(name)
}

func (this *Context) PostValues(name string) []string {
	if this.Request.PostForm == nil {
		this.WaitRequestData()
	}
	return this.Request.PostForm[name]
}

func (this *Context) Method() string {
	return this.Request.Method
}

func (this *Context) XMethod() string {
	xmeth := this.Request.Method

	if xmeth == "POST" {
		tmpxmeth := this.Request.Header.Get(REQMETHOD_X_METHOD_NAME)

		if tmpxmeth == "" {
			tmpxmeth = strings.ToUpper(this.PostValue(REQMETHOD_X_METHOD_NAME))
		}

		if validate.IsIn(tmpxmeth, "GET", "POST", "PUT", "PATCH", "DELETE") {
			xmeth = tmpxmeth
		}
	}

	return xmeth
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

func (this *Context) SetCookieValue(name, value string) {
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

func (this *Context) SetRedirect(location string, state ...int) int {
	if (strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://")) &&
		!strings.HasPrefix(stripSchema(location), stripSchema(this.URL(""))) {

		e := RedirectError{url: location}
		this.SetError(e.Error(), e.Code())
		return 0
	}

	return this.SetRedirectOut(location, state...)
}

func (this *Context) Redirect(location string, state ...int) {
	if this.redirecting.code == 0 {
		state := this.SetRedirect(location, state...)
		if state == 0 {
			panic(RedirectError{url: location})
		}
	}

	panic(this.redirecting)
}

func (this *Context) SetRedirectOut(location string, state ...int) (status int) {
	if this.redirecting.code == 0 {
		if !strings.HasPrefix(location, "http://") && !strings.HasPrefix(location, "https://") {
			location = this.URL(location)
		}

		status = 303

		if len(state) > 0 && state[0] > 0 {
			status = state[0]
		}

		http.Redirect(this.Response, this.Request, location, status)
		this.redirecting = Redirection{code: status, url: location}
	} else {
		status = -1
	}

	return
}

func (this *Context) RedirectOut(location string, state ...int) {
	if this.redirecting.code == 0 {
		this.SetRedirectOut(location, state...)
	}

	panic(this.redirecting)
}

func (this *Context) URL(path string, qstr ...Q) string {
	var buf bytes.Buffer

	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
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
		upath := this.Request.RequestURI
		if upath == "/" {
			upath = ""
		} else if upath != "" {
			if strings.Contains(upath, "?") {
				upath = strings.TrimSpace(strings.SplitN(upath, "?", 2)[0])
			}
			if upath[len(upath)-1] == '/' {
				upath = upath[0 : len(upath)-1]
			}
			if upath[0] != '/' {
				upath = "/" + upath
			}
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

func (this *Context) URLQ(path string, s ...string) string {
	return this.URL(path, this.QueryStr(s...))
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

func (this *Context) ToWebSocket(upgrader *websocket.Upgrader, header http.Header) (err error) {
	if upgrader == nil {
		upgrader = &websocket.Upgrader{}
		upgrader.CheckOrigin = func(_ *http.Request) bool {
			return true
		}
		upgrader.Error = func(w http.ResponseWriter, r *http.Request, status int, err error) {
			log.Println(err)
			http.Error(w, http.StatusText(status), status)
		}
	}

	this.WebSocket, err = upgrader.Upgrade(this.RootResponse(), this.Request, header)
	return
}

func (this *Context) RootResponse() (resp http.ResponseWriter) {
	resp = this.Response

	for {
		if cwriter, ok := resp.(ResponseWriterChild); ok {
			resp = cwriter.ParentResponseWriter()
			continue
		}
		break
	}

	return
}

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
	this.rdata = nil
	if this.WebSocket != nil {
		this.WebSocket.Close()
		this.WebSocket = nil
	}
}

func makeRequestData(k string, c map[string]interface{}, f url.Values) {
	if k != "" {
		k = k + "."
	}

	for _k, _v := range c {
		key := k + _k
		switch _s := _v.(type) {
		case []interface{}:
			for i, v := range _s {
				switch s := v.(type) {
				case string:
					f.Add(key, s)
				case json.Number:
					f.Add(key, s.String())
				case bool:
					f.Add(key, strconv.FormatBool(s))
				case map[string]interface{}:
					if key != "" {
						key = fmt.Sprintln("%s.%d", key, i)
					}
					makeRequestData(key, s, f)
				}
			}
		case string:
			f.Add(key, _s)
		case json.Number:
			f.Add(key, _s.String())
		case bool:
			f.Add(key, strconv.FormatBool(_s))
		case map[string]interface{}:
			makeRequestData(key, _s, f)
		}
	}
}

func requestDataToUrlValues(r *http.Request, data map[string]interface{}) (err error) {
	var (
		form        = make(url.Values)
		postform    = make(url.Values)
		queryValues url.Values
	)

	makeRequestData("", data, postform)

	for k, v := range postform {
		form[k] = append(form[k], v...)
	}

	if queryValues, err = url.ParseQuery(r.URL.RawQuery); err == nil {
		for k, v := range queryValues {
			form[k] = append(form[k], v...)
		}
	}

	r.PostForm = postform
	r.Form = form

	return
}

func stripSchema(url string) string {
	if strings.HasPrefix(url, "http://") {
		return url[7:]
	} else if strings.HasPrefix(url, "https://") {
		return url[8:]
	}

	return url
}
