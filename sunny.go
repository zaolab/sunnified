package sunnified

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zaolab/sunnified/config"
	"github.com/zaolab/sunnified/handler"
	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/mware"
	"github.com/zaolab/sunnified/router"
	"github.com/zaolab/sunnified/util/event"
	"github.com/zaolab/sunnified/web"
)

const ReqTimeout time.Duration = 10 * 60 * 1000 * 1000 * 1000 // 10mins
const DefaultMaxFileSize int64 = 26214400                     // 25MB

var (
	mutex   sync.RWMutex
	servers = make([]*SunnyApp, 0, 1)
)

type SunnyResponseWriter struct {
	http.ResponseWriter
	Status   int
	midwares []func(*web.Context)
	written  bool
	ctxt     *web.Context
}

func (sw *SunnyResponseWriter) WriteHeader(status int) {
	if !sw.written {
		sw.written = true
		defer func() {
			if err := recover(); err != nil {
				log.Println(err)
			}
			sw.Status = status
			sw.ResponseWriter.WriteHeader(status)
			sw.ctxt = nil
		}()
		for i := len(sw.midwares) - 1; i >= 0; i-- {
			sw.midwares[i](sw.ctxt)
		}
	}
}

func (sw *SunnyResponseWriter) Write(b []byte) (int, error) {
	if !sw.written {
		sw.WriteHeader(200)
	}
	return sw.ResponseWriter.Write(b)
}

func (sw *SunnyResponseWriter) ParentResponseWriter() http.ResponseWriter {
	return sw.ResponseWriter
}

type SunnyApp struct {
	router.Router
	id          int
	MiddleWares []mware.MiddleWare
	MaxFileSize int64
	conf        config.Library
	runners     int32
	closed      int32
	_callback   func()
	mutex       sync.Mutex
	ev          *event.Router
	controllers *controller.Group
	ctrlhand    *handler.DynamicHandler
	resources   map[string]func() interface{}
	mwareresp   []func(*web.Context)
	listener    net.Listener
}

func (sk *SunnyApp) Run(params map[string]interface{}) {
	var laddr = ":80"
	var timeout = ReqTimeout

	if dev, ok := params["dev"]; ok && dev.(bool) {
		laddr = "127.0.0.1:8080"
	} else {
		if port, ok := params["port"]; ok {
			var p = "80"
			switch v := port.(type) {
			case int:
				if v >= 1 && v <= 65535 {
					p = strconv.Itoa(v)
				}
			case int64:
				if v >= 1 && v <= 65535 {
					p = strconv.Itoa(int(v))
				}
			case float32:
				if v >= 1 && v <= 65535 {
					p = strconv.Itoa(int(v))
				}
			case float64:
				if v >= 1 && v <= 65535 {
					p = strconv.Itoa(int(v))
				}
			case string:
				if pint, err := strconv.Atoi(v); err == nil && pint >= 1 && pint <= 65535 {
					p = v
				}
			}
			laddr = ":" + p
		}
		if ip, ok := params["ip"]; ok {
			laddr = ip.(string) + laddr
		}
	}

	if tout, ok := params["timeout"]; ok {
		timeout = tout.(time.Duration)
	}

	if graceful, ok := params["graceful"]; ok && graceful.(bool) {
		GracefulShutDown()
	}

	if fastcgi, ok := params["fcgi"]; ok && fastcgi.(bool) {
		var err error

		if sock, ok := params["sock"]; ok && sock.(bool) {
			sockfile := "/tmp/sunnyapp.sock"
			if sfile, ok := params["sockfile"]; ok {
				sockfile = sfile.(string)
			}
			if _, err := os.Stat(sockfile); !os.IsNotExist(err) {
				log.Panicln("Error: socket file already in use. " + sockfile)
			}
			sk.listener, err = net.Listen("unix", sockfile)
			log.Println("Starting SunnyApp (FastCGI) on " + sockfile)
			GracefulShutDown()
		} else {
			sk.listener, err = net.Listen("tcp", laddr)
			log.Println("Starting SunnyApp (FastCGI) on " + laddr)
		}

		if err != nil {
			log.Panicln(err)
		}

		fcgi.Serve(sk.listener, sk)
	} else {
		log.Println("Starting SunnyApp on " + laddr)

		if err := newHTTPServer(laddr, sk, timeout).ListenAndServe(); err != nil {
			log.Panicln(err)
		}
	}
}

func (sk *SunnyApp) RunWithConfigFile(f string) {
	var cfg config.Configuration
	var err error
	if cfg, err = config.NewConfigurationFromFile(f); err != nil {
		log.Panicln(err)
	}

	sk.AddResourceFunc("sunnyconfig", func() interface{} { return cfg })

	if serverconf := cfg.Branch("server"); serverconf != nil {
		sk.Run(serverconf.ToMap())
	} else {
		sk.Run(cfg.ToMap())
	}
}

func (sk *SunnyApp) ID() int {
	return sk.id
}

func (sk *SunnyApp) IsClosed() bool {
	return atomic.LoadInt32(&sk.closed) == 1
}

func (sk *SunnyApp) AddMiddleWare(mwarecon mware.MiddleWare) {
	sk.MiddleWares = append(sk.MiddleWares, mwarecon)
	sk.mwareresp = append(sk.mwareresp, mwarecon.Response)
}

func (sk *SunnyApp) AddController(cinterface interface{}) {
	sk.controllers.AddController(cinterface)
	sk.createDynamicHandler()
}

func (sk *SunnyApp) SetControllerDefaults(action, control, mod string) {
	sk.createDynamicHandler()
	sk.ctrlhand.SetAction(action)
	sk.ctrlhand.SetController(control)
	sk.ctrlhand.SetModule(mod)
}

func (sk *SunnyApp) SetControllerDefaultAction(action string) {
	sk.createDynamicHandler()
	sk.ctrlhand.SetAction(action)
}

func (sk *SunnyApp) SetControllerDefaultControl(control string) {
	sk.createDynamicHandler()
	sk.ctrlhand.SetController(control)
}

func (sk *SunnyApp) SetControllerDefaultModule(mod string) {
	sk.createDynamicHandler()
	sk.ctrlhand.SetModule(mod)
}

func (sk *SunnyApp) SubRouter(name string) (rt router.Router) {
	rt = NewSunnyApp()
	sk.AddRouter(name, rt)
	return rt
}

func (sk *SunnyApp) AddResourceFunc(name string, f func() interface{}) {
	if sk.resources == nil {
		sk.resources = make(map[string]func() interface{})
	}
	sk.resources[name] = f
}

func (sk *SunnyApp) createDynamicHandler() {
	if sk.ctrlhand == nil {
		sk.ctrlhand = handler.NewDynamicHandler(sk.controllers)
		sk.Router.Handle("/{module*}/{controller*}/{action*}/", sk.ctrlhand)
	}
}

func (sk *SunnyApp) callback() {
	sk.mutex.Lock()
	defer sk.mutex.Unlock()
	if sk._callback != nil {
		sk._callback()
		sk._callback = nil
	}
}

func (sk *SunnyApp) triggererror(sunctxt *web.Context, err interface{}) {
	sk.triggerevent(sunctxt, "error", map[string]interface{}{"sunny.error": err})
	if sunctxt != nil {
		handler.InternalServerError(sunctxt.Response, sunctxt.Request)
	}
}

func (sk *SunnyApp) triggerevent(sunctxt *web.Context, eventname string, info map[string]interface{}) {
	if sunctxt != nil && sunctxt.Event != nil {
		sunctxt.Event.CreateTrigger("sunny").Fire(eventname, info)
	} else if sk.ev != nil {
		if info == nil {
			info = make(map[string]interface{})
		}
		info["sunny.context"] = nil
		sk.ev.CreateTrigger("sunny").Fire(eventname, info)
	}
}

func (sk *SunnyApp) decrunners() {
	if runners := atomic.AddInt32(&sk.runners, -1); runners == 0 && atomic.LoadInt32(&sk.closed) == 1 {
		removeSunnyApp(sk.id)
		sk.callback()
	}
}

func (sk *SunnyApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt, rep := sk.Router.FindRequestedEndPoint(make(map[string]interface{}), r)
	if rt == sk.Router {
		rt = sk
	}
	rt.ServeRequestedEndPoint(w, r, rep)
}

func (sk *SunnyApp) FindRequestedEndPoint(value map[string]interface{}, r *http.Request) (rt router.Router, rep *router.RequestedEndPoint) {
	rt, rep = sk.Router.FindRequestedEndPoint(value, r)
	if rt == sk.Router {
		rt = sk
	}
	return
}

func (sk *SunnyApp) ServeRequestedEndPoint(w http.ResponseWriter, r *http.Request, rep *router.RequestedEndPoint) {
	atomic.AddInt32(&sk.runners, 1)
	defer sk.decrunners()

	if atomic.LoadInt32(&sk.closed) == 1 || w == nil || r == nil {
		return
	}

	sw := &SunnyResponseWriter{
		Status:         200,
		ResponseWriter: w,
		midwares:       nil,
	}
	w = sw

	var sunctxt *web.Context

	if rep == nil {
		goto notfound
	}

	defer func() {
		if err := recover(); err != nil {
			if re, ok := err.(web.Redirection); ok {
				sk.triggerevent(sunctxt, "redirect", map[string]interface{}{"redirection": re})
			} else if e, ok := err.(web.ContextError); ok {
				sk.triggerevent(sunctxt, "contexterror", map[string]interface{}{"context.error": e})
				handler.ErrorHTML(w, r, e.Code())
			} else {
				log.Println(err)
				sk.triggererror(sunctxt, err)
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

	sunctxt = web.NewSunnyContext(w, r, sk.id)
	sunctxt.Event = sk.ev.NewSubRouter(event.M{"sunny.context": sunctxt})
	sunctxt.UPath = rep.UPath
	sunctxt.PData = rep.PData
	sunctxt.Ext = rep.Ext
	sunctxt.MaxFileSize = sk.MaxFileSize
	sunctxt.ParseRequestData()
	sw.ctxt = sunctxt

	for n, f := range sk.resources {
		sunctxt.SetResource(n, f())
	}

	for _, midware := range sk.MiddleWares {
		midware.Request(sunctxt)
		defer midware.Cleanup(sunctxt)
	}

	if router.HandleHeaders(sunctxt, rep.EndPoint, rep.Handler) {
		return
	}

	sw.midwares = sk.mwareresp
	for _, midware := range sk.MiddleWares {
		midware.Body(sunctxt)
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
		for _, midware := range sk.MiddleWares {
			midware.Controller(sunctxt, ctrlmgr)
		}

		state, vw := ctrlmgr.PrepareAndExecute()
		defer ctrlmgr.Cleanup()

		if vw != nil && !sunctxt.IsRedirecting() && !sunctxt.HasError() {
			setFuncMap(sunctxt, vw)

			// TODO: View should not matter which is called first..
			// make it a goroutine once determined sunctxt and ctrlmgr is completely thread-safe
			for _, midware := range sk.MiddleWares {
				midware.View(sunctxt, vw)
			}

			if err := ctrlmgr.PublishView(); err != nil {
				log.Println(err)
			}
		}

		if sunctxt.HasError() {
			sk.triggerevent(sunctxt, "contexterror", map[string]interface{}{"context.error": sunctxt.AppError()})
			handler.ErrorHTML(w, r, sunctxt.ErrorCode())
		} else if sunctxt.IsRedirecting() {
			sk.triggerevent(sunctxt, "redirect", map[string]interface{}{"redirection": sunctxt.Redirection()})
		} else if state != -1 && (state < 200 || state >= 300) {
			handler.ErrorHTML(w, r, state)
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

func (sk *SunnyApp) Close(callback func()) bool {
	sk.mutex.Lock()
	if callback != nil {
		sk._callback = callback
	}
	sk.mutex.Unlock()

	if atomic.CompareAndSwapInt32(&sk.closed, 0, 1) {
		if atomic.AddInt32(&sk.runners, -1) == 0 {
			removeSunnyApp(sk.id)
			sk.callback()
		}
		return true
	}

	return false
}

func (sk *SunnyApp) clear() {
	sk.ev = nil
	sk.MiddleWares = nil
	if sk.listener != nil {
		sk.listener.Close()
		sk.listener = nil
	}
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
		MaxFileSize: DefaultMaxFileSize,
		controllers: controller.NewControllerGroup(),
		runners:     1,
		mwareresp:   make([]func(*web.Context), 0, 5),
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

func removeSunnyApp(id int) {
	mutex.Lock()
	defer mutex.Unlock()
	if id > 0 && len(servers) > id && servers[id] != nil {
		servers[id].ev.CreateTrigger("sunny").Fire("shutdown", nil)
		servers[id].clear()
		servers[id] = nil
	}
}

func newHTTPServer(addr string, handler http.Handler, timeout time.Duration) *http.Server {
	return &http.Server{Addr: addr, Handler: handler, ReadTimeout: timeout}
}
