package sunnified

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zaolab/sunnified/config"
	"github.com/zaolab/sunnified/handler"
	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/mware"
	"github.com/zaolab/sunnified/router"
	"github.com/zaolab/sunnified/util/event"
	"github.com/zaolab/sunnified/util/validate"
	"github.com/zaolab/sunnified/web"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const REQ_TIMEOUT time.Duration = 10 * 60 * 1000 * 1000 * 1000 // 10mins
const DEFAULT_MAX_FILESIZE int64 = 104857600                   // 100MB

var (
	mutex   sync.RWMutex
	servers []*SunnyApp = make([]*SunnyApp, 0, 1)
)

type SunnyResponseWriter struct {
	Status int
	http.ResponseWriter
}

func (this *SunnyResponseWriter) WriteHeader(status int) {
	this.Status = status
	this.ResponseWriter.WriteHeader(status)
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
	if sunctxt != nil && sunctxt.Event != nil {
		sunctxt.Event.CreateTrigger("sunny").Fire("error500", map[string]interface{}{"sunny.error": err})
	} else if this.ev != nil {
		this.ev.CreateTrigger("sunny").Fire("error500", map[string]interface{}{"sunny.context": nil, "sunny.error": err})
	}
	handler.InternalServerError(sunctxt.Response, sunctxt.Request)
}

func (this *SunnyApp) parsereq(w http.ResponseWriter, r *http.Request) (waitr chan error) {
	waitr = make(chan error, 1)

	if validate.IsIn(r.Method, "POST", "PUT", "DELETE") {
		if this.MaxFileSize > 0 && strings.ToLower(r.Header.Get("Expect")) == "100-continue" &&
			r.ContentLength != -1 && r.ContentLength > this.MaxFileSize {

			w.Header().Set("Connection", "close")
			w.WriteHeader(http.StatusExpectationFailed)
			return nil
		}

		r.Body = http.MaxBytesReader(w, r.Body, this.MaxFileSize)
		go func() {
			var err error
			defer func() {
				if panicerr := recover(); panicerr != nil {
					log.Println(panicerr, err)
					waitr <- errors.New("Form parsing exited with panic")
				} else {
					waitr <- err
				}
			}()

			// angularjs post with application/json content-type by default
			if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json;") {
				var content map[string]interface{}
				var f = make(url.Values)
				err = json.NewDecoder(r.Body).Decode(&content)

				for k, v := range content {
					if slice, ok := v.([]string); ok {
						for _, s := range slice {
							f.Add(k, s)
						}
					} else if s, ok := v.(string); ok {
						f.Add(k, s)
					}
				}

				if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
					r.PostForm = f
					r.Form = make(url.Values)
					for k, v := range r.PostForm {
						r.Form[k] = append(r.Form[k], v...)
					}
				} else {
					r.Form = f
					r.PostForm = make(url.Values)
				}

				var queryValues url.Values
				if queryValues, err = url.ParseQuery(r.URL.RawQuery); err == nil {
					for k, v := range queryValues {
						r.Form[k] = append(r.Form[k], v...)
					}
				}
			} else {
				// 2MB in memory
				// passing back of ErrNotMultipart is only >= golang1.3
				if err = r.ParseMultipartForm(2097152); err == http.ErrNotMultipart {
					err = nil
				} else if (r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH") && err == nil && r.MultipartForm != nil {
					for k, v := range r.MultipartForm.Value {
						r.PostForm[k] = append(r.PostForm[k], v...)
					}
				}
			}
		}()
	} else {
		waitr <- nil
	}

	return
}

func (this *SunnyApp) decrunners() {
	atomic.AddInt32(&this.runners, -1)
	if atomic.LoadInt32(&this.runners) == 0 && atomic.LoadInt32(&this.closed) == 1 {
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
	if atomic.LoadInt32(&this.closed) == 1 {
		return
	}

	atomic.AddInt32(&this.runners, 1)
	defer this.decrunners()

	w = &SunnyResponseWriter{
		Status:         200,
		ResponseWriter: w,
	}

	var waitr chan error
	var sunctxt *web.Context

	if rep == nil {
		goto notfound
	}

	sunctxt = web.NewSunnyContext(w, r, this.id)
	sunctxt.Event = this.ev.NewSubRouter(event.M{"sunny.context": sunctxt})
	sunctxt.UPath = rep.UPath
	sunctxt.PData = rep.PData
	sunctxt.Ext = rep.Ext

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			this.triggererror(sunctxt, err)
		}
		log.Println(fmt.Sprintf("ip: %s; r: %s %s; d: %s; %d",
			sunctxt.RemoteAddress().String(), r.Method, r.URL.Path, time.Since(sunctxt.StartTime()).String(),
			w.(*SunnyResponseWriter).Status))
		sunctxt.Close()
		if r.MultipartForm != nil {
			r.MultipartForm.RemoveAll()
		}
	}()

	// all functions should not attempt to read form data
	// until waitr is over at controller
	if waitr = this.parsereq(w, r); waitr == nil {
		return
	}
	defer close(waitr)

	for _, midware := range this.MiddleWares {
		midware.Request(sunctxt)
		defer midware.Cleanup(sunctxt)
	}

	if router.HandleHeaders(sunctxt, rep.EndPoint, rep.Handler) {
		return
	}

	for _, midware := range this.MiddleWares {
		midware.Body(sunctxt)
		defer midware.Response(sunctxt)
	}

	if ctrl, ok := rep.Handler.(controller.ControlHandler); ok {
		ctrlmgr := ctrl.GetControlManager(sunctxt)

		if ctrlmgr == nil {
			goto notfound
		}

		// all request should default to no cache
		// TODO: create a middleware for this instead...
		//if r.Method != "HEAD" && w.Header().Get("Cache-Control") == "" {
		//	sunctxt.PrivateNoCache()
		//}

		sunctxt.Module = ctrlmgr.ModuleName()
		sunctxt.Controller = ctrlmgr.ControllerName()
		sunctxt.Action = ctrlmgr.ActionName()

		if err := <-waitr; err != nil {
			setreqerror(err, w)
			return
		}

		func() {
			defer func() {
				if err := recover(); err != nil {
					switch e := err.(type) {
					case web.RedirectError:
					case web.AppError:
						handler.ErrorHtml(w, r, e.Code())
					default:
						panic(e)
					}
				}
			}()

			// TODO: Controller should not matter which is called first..
			// make it a goroutine once determined sunctxt and ctrlmgr is completely thread-safe
			for _, midware := range this.MiddleWares {
				midware.Controller(sunctxt, ctrlmgr)
			}

			state, vw := ctrlmgr.PrepareAndExecute()

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
				handler.ErrorHtml(w, r, sunctxt.ErrorCode())
			} else if !sunctxt.IsRedirecting() && state != -1 && (state < 200 || state >= 300) {
				handler.ErrorHtml(w, r, state)
			}
		}()

		ctrlmgr.Cleanup()
	} else {
		if err := <-waitr; err != nil {
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
	// TODO: notify multiparser to stop parsing if it hasn't already
	handler.NotFound(w, r)
}

func (this *SunnyApp) Close(callback func()) bool {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if atomic.CompareAndSwapInt32(&this.closed, 0, 1) {
		if atomic.LoadInt32(&this.runners) == 0 && callback != nil {
			removeSunnyApp(this.id)
			callback()
		} else {
			this._callback = callback
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
