package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/zaolab/sunnified/util"
	"github.com/zaolab/sunnified/util/collection"
	"github.com/zaolab/sunnified/util/event"
	"github.com/zaolab/sunnified/util/validate"
)

type ResponseWriterChild interface {
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

func (c *Context) WaitRequestData() error {
	if !c.parseState.Started() {
		c.ParseRequestData()
	}

	c.parseState.Status.Wait()
	return c.parseState.Error
}

func (c *Context) parseRequestData(ctype string, body io.ReadCloser) {
	var err error

	defer func() {
		if panicerr := recover(); panicerr != nil {
			log.Println(panicerr, err)
			c.parseState.End(errors.New("form parsing exited with panic"))
		} else {
			c.parseState.End(err)
		}
	}()

	if strings.Contains(ctype, ";") {
		ctype = strings.TrimSpace(strings.SplitN(ctype, ";", 2)[0])
	}

	if parser := GetContentParser(ctype); parser != nil {
		c.rdata = parser(body)
		requestDataToUrlValues(c.Request, c.rdata)
	} else if ctype == "application/json" {
		d := json.NewDecoder(body)
		d.UseNumber()
		d.Decode(&c.rdata)
		requestDataToUrlValues(c.Request, c.rdata)
	} else {
		// 2MB in memory
		// passing back of ErrNotMultipart is only >= golang1.3
		if err = c.Request.ParseMultipartForm(2097152); err == http.ErrNotMultipart {
			err = nil
		} else if err == nil && c.Request.MultipartForm != nil {
			if c.Request.PostForm == nil {
				c.Request.PostForm = make(url.Values)
			}
			for k, v := range c.Request.MultipartForm.Value {
				c.Request.PostForm[k] = append(c.Request.PostForm[k], v...)
			}
		}
	}
}

func (c *Context) ParseRequestData() {
	if !c.parseState.Start() {
		return
	}

	var (
		r             = c.Request
		w             = c.Response
		maxsize int64 = 10485760 //10MB
	)

	if validate.IsIn(r.Method, "POST", "PUT", "PATCH") {
		ctype := strings.ToLower(r.Header.Get("Content-Type"))
		is100 := strings.ToLower(r.Header.Get("Expect")) == "100-continue"

		if !strings.HasPrefix(ctype, "multipart") {
			if is100 && r.ContentLength > maxsize {
				goto expectFailed
			}
		} else if c.MaxFileSize > 0 {
			if maxsize = c.MaxFileSize; is100 && r.ContentLength > maxsize {
				goto expectFailed
			}

			r.Body = http.MaxBytesReader(w, r.Body, maxsize)
		}

		go c.parseRequestData(ctype, r.Body)
	} else {
		var err error

		if c.Request.Form == nil {
			c.Request.Form, err = url.ParseQuery(c.Request.URL.RawQuery)
		}

		c.parseState.End(err)
	}

	return

expectFailed:
	e := ExpectationError{size: r.ContentLength, maxsize: maxsize}
	c.parseState.End(e)
	w.Header().Set("Connection", "close")
	panic(e)
}

func (c *Context) RequestBodyData(ctype string) map[string]interface{} {
	if ctype == "" || strings.HasPrefix(strings.ToLower(c.Request.Header.Get("Content-Type")), ctype) {
		return c.rdata
	}

	return nil
}

func (c *Context) RaiseAppError(err string, status ...int) {
	c.SetError(err, status...)
	panic(AppError{code: c.errorcode, err: err})
}

func (c *Context) SetErrorCode(status int) {
	c.errorcode = status
}

func (c *Context) ErrorCode() int {
	return c.errorcode
}

func (c *Context) SetError(err string, status ...int) {
	if len(status) > 0 {
		c.errorcode = status[0]
	} else {
		c.errorcode = 500
	}

	c.err = err
}

func (c *Context) HasErrorCode() bool {
	return c.HasError()
}

func (c *Context) HasError() bool {
	return c.errorcode != 200 && c.errorcode != 0
}

func (c *Context) Error() string {
	return c.err
}

func (c *Context) RecoverError() {
	c.errorcode = 0
	c.err = ""
}

func (c *Context) AppError() (ae AppError) {
	if c.HasError() {
		ae = AppError{code: c.errorcode, err: c.err}
	}

	return
}

func (c *Context) SetTitle(title string) {
	if c.SetTitle_ != nil {
		c.SetTitle_(title)
	}
}

func (c *Context) ReqHeader(header string) string {
	return c.Request.Header.Get(header)
}

func (c *Context) ReqHeaderHas(header, value string) bool {
	return strings.Contains(c.Request.Header.Get(header), value)
}

func (c *Context) ReqHeaderIs(header, value string) bool {
	return c.Request.Header.Get(header) == value
}

func (c *Context) ResHeader(header string) string {
	return c.Response.Header().Get(header)
}

func (c *Context) SetHeader(header, value string) {
	c.Response.Header().Set(header, value)
}

func (c *Context) AddHeader(header, value string) {
	c.Response.Header().Add(header, value)
}

func (c *Context) StartTime() time.Time {
	return c.time
}

func (c *Context) SunnyServerId() int {
	// to prevent mistaking of sunny server when Context created without
	// using NewContext
	if c.issunny {
		return c.sunnyserver
	} else {
		return -1
	}
}

func (c *Context) IsSunnyContext() bool {
	return c.issunny
}

func (c *Context) IsRedirecting() bool {
	return c.redirecting.code > 0
}

func (c *Context) IsHttps() bool {
	return c.IsHTTPS()
}

func (c *Context) IsHTTPS() bool {
	return c.Request.TLS != nil
}

func (c *Context) IsCors() bool {
	return c.IsCORS()
}

func (c *Context) IsCORS() bool {
	return c.Request.Header.Get("Origin") != ""
}

func (c *Context) IsAjax() bool {
	return c.IsAJAX()
}

func (c *Context) IsAJAX() bool {
	return strings.ToLower(c.Request.Header.Get(HTTP_X_REQUESTED_WITH)) == "xmlhttprequest"
}

func (c *Context) IsAjaxOrCors() bool {
	return c.IsAJAXOrCORS()
}

func (c *Context) IsAJAXOrCORS() bool {
	return c.IsAJAX() || c.IsCORS()
}

func (c *Context) RedirectionCode() int {
	return c.redirecting.code
}

func (c *Context) RedirectionURL() string {
	return c.redirecting.url
}

func (c *Context) Redirection() Redirection {
	return c.redirecting
}

func (c *Context) RemoteAddress() net.IP {
	raddr := c.Request.RemoteAddr
	if index := strings.Index(raddr, ":"); index != -1 {
		raddr = raddr[0:index]
	}

	return net.ParseIP(raddr)
}

func (c *Context) FwdedForOrRmteAddr() (ip net.IP) {
	if ipstr := c.Request.Header.Get(HTTP_X_FORWARDED_FOR); ipstr != "" {
		if cindex := strings.Index(ipstr, ","); cindex != -1 {
			ip = net.ParseIP(ipstr[0:cindex])
		} else {
			ip = net.ParseIP(ipstr)
		}
	}

	if ip == nil {
		ip = c.RemoteAddress()
	}
	return
}

func (c *Context) XRealIPOrRmteAddr() (ip net.IP) {
	if ipstr := c.Request.Header.Get(HTTP_X_REAL_IP); ipstr != "" {
		ip = net.ParseIP(ipstr)
	}

	if ip == nil {
		ip = c.RemoteAddress()
	}
	return
}

func (c *Context) ClientIP(pref []string) (ip net.IP) {
	if pref != nil {
		for _, h := range pref {
			if ipstr := c.Request.Header.Get(h); ipstr != "" {
				ip = net.ParseIP(ipstr)
				if ip != nil {
					return
				}
			}
		}
	}

	ip = c.RemoteAddress()
	return
}

func (c *Context) RequestValue(name string) string {
	if c.Request.Form == nil {
		c.WaitRequestData()
	}
	return c.Request.FormValue(name)
}

func (c *Context) RequestValues(name string) []string {
	if c.Request.Form == nil {
		c.WaitRequestData()
	}
	return c.Request.Form[name]
}

func (c *Context) PostValue(name string) string {
	if c.Request.PostForm == nil {
		c.WaitRequestData()
	}
	return c.Request.PostFormValue(name)
}

func (c *Context) PostValues(name string) []string {
	if c.Request.PostForm == nil {
		c.WaitRequestData()
	}
	return c.Request.PostForm[name]
}

func (c *Context) Method() string {
	return c.Request.Method
}

func (c *Context) XMethod() string {
	xmeth := c.Request.Method

	if xmeth == "POST" {
		tmpxmeth := c.Request.Header.Get(REQMETHOD_X_METHOD_NAME)

		if tmpxmeth == "" {
			tmpxmeth = strings.ToUpper(c.PostValue(REQMETHOD_X_METHOD_NAME))
		}

		if validate.IsIn(tmpxmeth, "GET", "POST", "PUT", "PATCH", "DELETE") {
			xmeth = tmpxmeth
		}
	}

	return xmeth
}

func (c *Context) SetETag(etag string) {
	c.Response.Header().Set("ETag", etag)
}

func (c *Context) SetCookie(ck *http.Cookie) {
	if c.IsHttps() && !ck.Secure {
		ck.Secure = true
	}
	http.SetCookie(c.Response, ck)
}

func (c *Context) SetCookieValue(name, value string) {
	c.SetCookie(&http.Cookie{
		Name:  name,
		Value: value,
	})
}

func (c *Context) DeleteCookie(cname string) {
	c.SetCookie(&http.Cookie{
		Name:   cname,
		MaxAge: -1,
	})
}

func (c *Context) Cookie(cname string) (ck *http.Cookie) {
	ck, _ = c.Request.Cookie(cname)
	return
}

func (c *Context) CookieValue(cname string) string {
	ck := c.Cookie(cname)
	if ck != nil {
		return ck.Value
	}
	return ""
}

func (c *Context) AddFlash(msg string) {
	if c.Session != nil {
		c.Session.AddFlash(msg)
	} else {
		if c.flashcache == nil {
			c.flashcache = collection.NewQueue(msg)
		} else {
			c.flashcache.Push(msg)
		}
	}
}

func (c *Context) HasFlash() bool {
	if c.Session != nil {
		return c.Session.HasFlash()
	} else {
		return c.flashcache != nil && c.flashcache.HasQueue()
	}
}

func (c *Context) Flash() string {
	if c.Session != nil {
		return c.Session.Flash()
	} else if c.flashcache != nil {
		return c.flashcache.PullDefault("").(string)
	}
	return ""
}

func (c *Context) AllFlashes() []string {
	if c.Session != nil {
		return c.Session.AllFlashes()
	} else if c.flashcache != nil {
		flashes := c.PeekFlashes()
		c.flashcache.Clear()
		return flashes
	}

	return nil
}

func (c *Context) PeekFlashes() []string {
	if c.Session != nil {
		return c.Session.PeekFlashes()
	} else if c.flashcache != nil {
		var flash string
		flashes := make([]string, 0, c.flashcache.Len())

		for iter := c.flashcache.Iterator(); iter.Next(&flash); {
			flashes = append(flashes, flash)
		}

		return flashes
	}

	return nil
}

func (c *Context) LenFlashes() int {
	if c.Session != nil {
		return c.Session.LenFlashes()
	} else if c.flashcache != nil {
		return c.flashcache.Len()
	}
	return 0
}

func (c *Context) PrivateNoStore() {
	header := c.Response.Header()
	header.Set("Pragma", "no-cache")
	header.Set("Cache-Control", "private, max-age=0, no-cache, no-store")
}

func (c *Context) PrivateNoCache() {
	header := c.Response.Header()
	header.Set("Pragma", "no-cache")
	header.Set("Cache-Control", "private, max-age=0, no-cache")
}

func (c *Context) PublicCache(age int) {
	header := c.Response.Header()

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

func (c *Context) SetRedirect(location string, state ...int) int {
	if (strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://")) &&
		!strings.HasPrefix(stripSchema(location), stripSchema(c.URL(""))) {

		e := RedirectError{url: location}
		c.SetError(e.Error(), e.Code())
		return 0
	}

	return c.SetRedirectOut(location, state...)
}

func (c *Context) Redirect(location string, state ...int) {
	if c.redirecting.code == 0 {
		state := c.SetRedirect(location, state...)
		if state == 0 {
			panic(RedirectError{url: location})
		}
	}

	panic(c.redirecting)
}

func (c *Context) SetRedirectOut(location string, state ...int) (status int) {
	if c.redirecting.code == 0 {
		if !strings.HasPrefix(location, "http://") && !strings.HasPrefix(location, "https://") {
			location = c.URL(location)
		}

		status = 303

		if len(state) > 0 && state[0] > 0 {
			status = state[0]
		}

		http.Redirect(c.Response, c.Request, location, status)
		c.redirecting = Redirection{code: status, url: location}
	} else {
		status = -1
	}

	return
}

func (c *Context) RedirectOut(location string, state ...int) {
	if c.redirecting.code == 0 {
		c.SetRedirectOut(location, state...)
	}

	panic(c.redirecting)
}

func (c *Context) URL(path string, qstr ...Q) string {
	var buf bytes.Buffer

	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		buf.WriteString("http")
		if c.Request.TLS != nil {
			buf.WriteByte('s')
		}
		buf.WriteString("://")
	}

	hasstar := strings.HasPrefix(path, "*")

	if host, pathl := strings.ToLower(c.Request.Host), strings.ToLower(path); !strings.Contains(pathl, host) {
		buf.WriteString(host)
		if !hasstar && pathl != "" && pathl[0] != '/' {
			buf.WriteString("/")
		}
	}

	if hasstar {
		upath := c.Request.RequestURI
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

func (c *Context) URLQ(path string, s ...string) string {
	return c.URL(path, c.QueryStr(s...))
}

func (c *Context) QueryStr(s ...string) (qs Q) {
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

func (c *Context) MapResourceValue(name string, ref interface{}) (err error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if val, ok := c.resource[name]; ok && ref != nil {
		return util.MapValue(ref, val)
	}

	return ErrResourceNotFound
}

func (c *Context) Resource(name string) (val interface{}) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	val, _ = c.resource[name]

	return
}

func (c *Context) SetResource(name string, ref interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.resource[name] = ref
}

func (c *Context) ToWebSocket(upgrader *websocket.Upgrader, header http.Header) (err error) {
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

	c.WebSocket, err = upgrader.Upgrade(c.RootResponse(), c.Request, header)
	return
}

func (c *Context) RootResponse() (resp http.ResponseWriter) {
	resp = c.Response

	for {
		if cwriter, ok := resp.(ResponseWriterChild); ok {
			resp = cwriter.ParentResponseWriter()
			continue
		}
		break
	}

	return
}

func (c *Context) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.resource = nil
	c.Session = nil
	c.Cache = nil
	c.Event = nil
	c.UPath = nil
	c.PData = nil
	c.Request = nil
	c.Response = nil
	c.SetTitle_ = nil
	c.rdata = nil
	if c.WebSocket != nil {
		c.WebSocket.Close()
		c.WebSocket = nil
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
