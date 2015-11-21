package sunnified

import (
	"fmt"
	"github.com/zaolab/sunnified/config"
	"github.com/zaolab/sunnified/handler"
	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/mware"
	"github.com/zaolab/sunnified/router"
	"github.com/zaolab/sunnified/util/event"
	"github.com/zaolab/sunnified/web"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"
)

const REQ_TIMEOUT time.Duration = 10 * 60 * 1000 * 1000 * 1000 // 10mins
const DEFAULT_MAX_FILESIZE int64 = 26214400                    // 25MB

var (
	mutex   sync.RWMutex
	servers []*SunnyApp = make([]*SunnyApp, 0, 1)
)

type SunnyResponseWriter struct {
	http.ResponseWriter
	Status   int
	midwares []func(*web.Context)
	written  bool
	ctxt     *web.Context
}

func (this *SunnyResponseWriter) WriteHeader(status int) {
	if !this.written {
		this.written = true
		defer func() {
			recover()
			this.Status = status
			this.ResponseWriter.WriteHeader(status)
		}()
		for _, mw := range this.midwares {
			mw(this.ctxt)
		}
	}
}

func (this *SunnyResponseWriter) Write(b []byte) (int, error) {
	if !this.written {
		this.WriteHeader(200)
	}
	return this.ResponseWriter.Write(b)
}

type SunnyApp struct {
	router.Router
	id          int
	MiddleWares []mware.MiddleWare
	MaxFileSize int64
	conf        config.ConfigLibrary
	runners     int32
	closed      int32
	_callback   func()
	mutex       sync.Mutex
	ev          *event.EventRouter
	controllers *controller.ControllerGroup
	ctrlhand    *handler.DynamicHandler
	resources   map[string]func() interface{}
}

func (this *SunnyApp) Run(params map[string]interface{}) {
	var laddr string = ":80"
	var timeout time.Duration = REQ_TIMEOUT

	if dev, ok := params["dev"]; ok && dev.(bool) {
		laddr = "127.0.0.1:8080"
	} else {
		if port, ok := params["port"]; ok {
			laddr = ":" + port.(string)
		}
		if ip, ok := params["ip"]; ok {
			laddr = ip.(string) + laddr
		}
	}

	if tout, ok := params["timeout"]; ok {
		timeout = tout.(time.Duration)
	}

	if fastcgi, ok := params["fcgi"]; ok && fastcgi.(bool) {
		var listener net.Listener
		var err error

		if sock, ok := params["sock"]; ok && sock.(bool) {
			sockfile := "/tmp/sunnyapp.sock"
			if sfile, ok := params["sockfile"]; ok {
				sockfile = sfile.(string)
			}
			listener, err = net.Listen("unix", sockfile)
			log.Println("Starting SunnyApp (FastCGI) on " + sockfile)
		} else {
			listener, err = net.Listen("tcp", laddr)
			log.Println("Starting SunnyApp (FastCGI) on " + laddr)
		}

		if err != nil {
			log.Panicln(err)
		}
		defer listener.Close()

		fcgi.Serve(listener, this)
	} else {
		log.Println("Starting SunnyApp on " + laddr)

		if err := newHttpServer(laddr, this, timeout).ListenAndServe(); err != nil {
			log.Panicln(err)
		}
	}
}

func (this *SunnyApp) Id() int {
	return this.id
}

func (this *SunnyApp) IsClosed() bool {
	return atomic.LoadInt32(&this.closed) == 1
}

func (this *SunnyApp) AddMiddleWare(mwarecon mware.MiddleWare) {
	this.MiddleWares = append(this.MiddleWares, mwarecon)
}

func (this *SunnyApp) AddController(cinterface interface{}) {
	this.controllers.AddController(cinterface)
	this.createDynamicHandler()
}

func (this *SunnyApp) SetControllerDefaults(action, control, mod string) {
	this.createDynamicHandler()
	this.ctrlhand.SetAction(action)
	this.ctrlhand.SetController(control)
	this.ctrlhand.SetModule(mod)
}

func (this *SunnyApp) SetControllerDefaultAction(action string) {
	this.createDynamicHandler()
	this.ctrlhand.SetAction(action)
}

func (this *SunnyApp) SetControllerDefaultControl(control string) {
	this.createDynamicHandler()
	this.ctrlhand.SetController(control)
}

func (this *SunnyApp) SetControllerDefaultModule(mod string) {
	this.createDynamicHandler()
	this.ctrlhand.SetModule(mod)
}

func (this *SunnyApp) SubRouter(name string) (rt router.Router) {
	rt = NewSunnyApp()
	this.AddRouter(name, rt)
	return rt
}

func (this *SunnyApp) AddResourceFunc(name string, f func() interface{}) {
	if this.resources == nil {
		this.resources = make(map[string]func() interface{})
	}
	this.resources[name] = f
}

func (this *SunnyApp) createDynamicHandler() {
	if this.ctrlhand == nil {
		this.ctrlhand = handler.NewDynamicHandler(this.controllers)
		this.Router.Handle("/{module*}/{controller*}/{action*}/", this.ctrlhand)
	}
}

func (this *SunnyApp) callback() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this._callback != nil {
		this._callback()
		this._callback = nil
	}
}

func (this *SunnyApp) triggererror(sunctxt *web.Context, err interface{}) {
	this.triggerevent(sunctxt, "error", map[string]interface{}{"sunny.error": err})
	if sunctxt != nil {
		handler.InternalServerError(sunctxt.Response, sunctxt.Request)
	}
}

func (this *SunnyApp) triggerevent(sunctxt *web.Context, eventname string, info map[string]interface{}) {
	if sunctxt != nil && sunctxt.Event != nil {
		sunctxt.Event.CreateTrigger("sunny").Fire(eventname, info)
	} else if this.ev != nil {
		if info == nil {
			info = make(map[string]interface{})
		}
		info["sunny.context"] = nil
		this.ev.CreateTrigger("sunny").Fire(eventname, info)
	}
}

func (this *SunnyApp) decrunners() {
	if runners := atomic.AddInt32(&this.runners, -1); runners == 0 && atomic.LoadInt32(&this.closed) == 1 {
		removeSunnyApp(this.id)
		this.callback()
	}
}

func (this *SunnyApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt, rep := this.Router.FindRequestedEndPoint(make(map[string]interface{}), r)
	if rt == this.Router {
		rt = this
	}
	rt.ServeRequestedEndPoint(w, r, rep)
}

func (this *SunnyApp) FindRequestedEndPoint(value map[string]interface{}, r *http.Request) (rt router.Router, rep *router.RequestedEndPoint) {
	rt, rep = this.Router.FindRequestedEndPoint(value, r)
	if rt == this.Router {
		rt = this
	}
	return
}

func (this *SunnyApp) ServeRequestedEndPoint(w http.ResponseWriter, r *http.Request, rep *router.RequestedEndPoint) {
	atomic.AddInt32(&this.runners, 1)
	defer this.decrunners()

	if atomic.LoadInt32(&this.closed) == 1 || w == nil || r == nil {
		return
	}

	sw := &SunnyResponseWriter{
		Status:         200,
		ResponseWriter: w,
		midwares:       make([]func(*web.Context), 0, len(this.MiddleWares)),
	}
	w = sw

	var sunctxt *web.Context

	if rep == nil {
		goto notfound
	}

	defer func() {
		if err := recover(); err != nil {
			if re, ok := err.(web.Redirection); ok {
				this.triggerevent(sunctxt, "redirect", map[string]interface{}{"redirection": re})
			} else if e, ok := err.(web.ContextError); ok {
				this.triggerevent(sunctxt, "contexterror", map[string]interface{}{"context.error": e})
				handler.ErrorHtml(w, r, e.Code())
			} else {
				log.Println(err)
				this.triggererror(sunctxt, err)
			}
		}

		if sunctxt != nil {
			log.Println(fmt.Sprintf("ip: %s; r: %s %s; d: %s; %d",
				sunctxt.RemoteAddress().String(), r.Method, r.URL.Path, time.Since(sunctxt.StartTime()).String(),
				w.(*SunnyResponseWriter).Status))
			sunctxt.Close()
		}
		if r.MultipartForm != nil {
			r.MultipartForm.RemoveAll()
		}
	}()

	sunctxt = web.NewSunnyContext(w, r, this.id)
	sunctxt.Event = this.ev.NewSubRouter(event.M{"sunny.context": sunctxt})
	sunctxt.UPath = rep.UPath
	sunctxt.PData = rep.PData
	sunctxt.Ext = rep.Ext
	sunctxt.MaxFileSize = this.MaxFileSize
	sunctxt.ParseRequestData()
	sw.ctxt = sunctxt

	for n, f := range this.resources {
		sunctxt.SetResource(n, f())
	}

	for _, midware := range this.MiddleWares {
		midware.Request(sunctxt)
		defer midware.Cleanup(sunctxt)
	}

	if router.HandleHeaders(sunctxt, rep.EndPoint, rep.Handler) {
		return
	}

	for _, midware := range this.MiddleWares {
		midware.Body(sunctxt)
		sw.midwares = append(sw.midwares, midware.Response)
	}

	if ctrl, ok := rep.Handler.(controller.ControlHandler); ok {
		ctrlmgr := ctrl.GetControlManager(sunctxt)

		if ctrlmgr == nil {
			goto notfound
		}

		sunctxt.Module = ctrlmgr.ModuleName()
		sunctxt.Controller = ctrlmgr.ControllerName()
		sunctxt.Action = ctrlmgr.ActionName()

		if err := sunctxt.WaitRequestData(); err != nil {
			setreqerror(err, w)
			return
		}

		// TODO: Controller should not matter which is called first..
		// make it a goroutine once determined sunctxt and ctrlmgr is completely thread-safe
		for _, midware := range this.MiddleWares {
			midware.Controller(sunctxt, ctrlmgr)
		}

		state, vw := ctrlmgr.PrepareAndExecute()
		defer ctrlmgr.Cleanup()

		if vw != nil && !sunctxt.IsRedirecting() && !sunctxt.HasError() {
			setFuncMap(sunctxt, vw)

			// TODO: View should not matter which is called first..
			// make it a goroutine once determined sunctxt and ctrlmgr is completely thread-safe
			for _, midware := range this.MiddleWares {
				midware.View(sunctxt, vw)
			}

			if err := ctrlmgr.PublishView(); err != nil {
				log.Println(err)
			}
		}

		if sunctxt.HasError() {
			this.triggerevent(sunctxt, "contexterror", map[string]interface{}{"context.error": sunctxt.AppError()})
			handler.ErrorHtml(w, r, sunctxt.ErrorCode())
		} else if sunctxt.IsRedirecting() {
			this.triggerevent(sunctxt, "redirect", map[string]interface{}{"redirection": sunctxt.Redirection()})
		} else if state != -1 && (state < 200 || state >= 300) {
			handler.ErrorHtml(w, r, state)
		}
	} else {
		if err := sunctxt.WaitRequestData(); err != nil {
			setreqerror(err, w)
			return
		}

		if h, ok := rep.Handler.(web.ContextHandler); ok {
			h.ServeContextHTTP(sunctxt)
		} else {
			rep.Handler.ServeHTTP(w, r)
		}
	}

	return

notfound:
	handler.NotFound(w, r)
}

func (this *SunnyApp) Close(callback func()) bool {
	this.mutex.Lock()
	if callback != nil {
		this._callback = callback
	}
	this.mutex.Unlock()

	if atomic.CompareAndSwapInt32(&this.closed, 0, 1) {
		if atomic.AddInt32(&this.runners, -1) == 0 {
			removeSunnyApp(this.id)
			this.callback()
		}
		return true
	}

	return false
}

func (this *SunnyApp) clear() {
	this.ev = nil
	this.MiddleWares = nil
}

func setreqerror(err error, w http.ResponseWriter) {
	if err.Error() == "http: request body too large" {
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
	} else if e, ok := err.(net.Error); ok && e.Timeout() {
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusRequestTimeout)
	} else {
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func NewSunnyApp() (ss *SunnyApp) {
	ss = &SunnyApp{
		Router:      router.NewSunnyRouter(),
		MiddleWares: make([]mware.MiddleWare, 0, 5),
		MaxFileSize: DEFAULT_MAX_FILESIZE,
		controllers: controller.NewControllerGroup(),
		runners:     1,
	}

	mutex.Lock()
	defer mutex.Unlock()
	ssid := len(servers)
	ss.id = ssid
	ss.ev = event.NewEventRouter(event.M{"sunny.id": ssid})
	servers = append(servers, ss)

	return
}

func GetSunnyApp(id int) *SunnyApp {
	mutex.RLock()
	defer mutex.RUnlock()
	if id > 0 && len(servers) > id {
		return servers[id]
	}
	return nil
}

var graceshut int32 = 0

func GracefulShutDown() {
	if atomic.CompareAndSwapInt32(&graceshut, 0, 1) {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

		go func() {
			<-c
			mutex.RLock()
			var serverslen = len(servers)
			var allservers = make([]*SunnyApp, serverslen)
			copy(allservers, servers)
			mutex.RUnlock()

			w := &sync.WaitGroup{}
			w.Add(serverslen)

			for _, server := range allservers {
				if !server.Close(func() { w.Done() }) {
					w.Done()
				}
			}

			w.Wait()
			os.Exit(1)
		}()
	}
}

func removeSunnyApp(id int) {
	mutex.Lock()
	defer mutex.Unlock()
	if id > 0 && len(servers) > id && servers[id] != nil {
		servers[id].ev.CreateTrigger("sunny").Fire("shutdown", nil)
		servers[id].clear()
		servers[id] = nil
	}
}

func newHttpServer(addr string, handler http.Handler, timeout time.Duration) *http.Server {
	return &http.Server{Addr: addr, Handler: handler, ReadTimeout: timeout}
}
