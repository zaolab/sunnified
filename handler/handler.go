package handler

import (
	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/router"
	"github.com/zaolab/sunnified/web"
	"net/http"
	"strings"
	"sync"
)

type DynamicHandler struct {
	module     string
	controller string
	action     string
	mutex      sync.RWMutex
	ctrlgroup  *controller.ControllerGroup
}

func NewDynamicHandler(ctrlgroup *controller.ControllerGroup) *DynamicHandler {
	if ctrlgroup == nil {
		ctrlgroup = controller.GetDefaultControllerGroup()
	}

	return &DynamicHandler{
		ctrlgroup: ctrlgroup,
	}
}

func NewDefaultDynamicHandler(ctrlgroup *controller.ControllerGroup, action, control, mod string) *DynamicHandler {
	if ctrlgroup == nil {
		ctrlgroup = controller.GetDefaultControllerGroup()
	}

	return &DynamicHandler{
		module:     mod,
		controller: control,
		action:     action,
		ctrlgroup:  ctrlgroup,
	}
}

func (this *DynamicHandler) SetModule(mod string) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.module = mod
}

func (this *DynamicHandler) SetController(control string) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.controller = control
}

func (this *DynamicHandler) SetAction(action string) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.action = action
}

func (this *DynamicHandler) Module() string {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.module
}

func (this *DynamicHandler) Controller() string {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.controller
}

func (this *DynamicHandler) Action() string {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.action
}

func (this *DynamicHandler) ServeOptions(w http.ResponseWriter, r *http.Request, origin map[string]string) {
	this.ServeContextOptions(web.NewContext(w, r), origin)
}

func (this *DynamicHandler) ServeContextOptions(context *web.Context, origin map[string]string) {
	router.SetHeaderOrigin(context.Response, context.Request, origin)

	var (
		header  http.Header                = context.Response.Header()
		ctrlmgr *controller.ControlManager = this.GetControlManager(context)
		mlist   []string                   = ctrlmgr.AvailableMethodsList()
	)

	mliststr := strings.Join(mlist, ", ")
	allow := "HEAD, OPTIONS"

	if mliststr != "" {
		allow += ", " + mliststr
	}

	header.Set("Allow", allow)
	header.Set("Access-Control-Allow-Methods", mliststr)
	context.Response.WriteHeader(200)
}

func (this *DynamicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	this.ServeContextHTTP(web.NewContext(w, r))
}

func (this *DynamicHandler) ServeContextHTTP(context *web.Context) {
	if ctrlmgr := this.GetControlManager(context); ctrlmgr != nil {
		ctrlmgr.Prepare()
		state, vw := ctrlmgr.Execute()
		if state != -1 && vw == nil {
			ErrorHtml(context.Response, context.Request, state)
			return
		}
		ctrlmgr.PublishView()
		ctrlmgr.Cleanup()
	} else {
		NotFound(context.Response, context.Request)
	}
}

func (this *DynamicHandler) GetControlManager(context *web.Context) (cm *controller.ControlManager) {
	this.mutex.RLock()
	var mod, control, act string = this.module, this.controller, this.action
	this.mutex.RUnlock()

	var acterr error
	var action string

	// if mod is not found, the default mod will be used
	// this allows user to use shortcut path
	// especially when there is only one mod
	if s, err := context.PData.String("module"); err == nil {
		if this.ctrlgroup.HasModule(s) || !this.ctrlgroup.HasController(mod, s) {
			mod = s
		} else {
			// controller.HasController(mod, s) == true
			// move {action} to UPath, {controller} to {action}, {mod} to {controller}
			if tmpact, err := context.PData.String("action"); err == nil {
				lenupath := len(context.UPath) + 1
				tmpupath := make(web.UPath, lenupath)
				tmpupath[0] = tmpact
				copy(tmpupath[1:lenupath], context.UPath)
				context.UPath = tmpupath
			}
			if tmpctrl, err := context.PData.String("controller"); err == nil {
				context.PData["action"] = tmpctrl
			}

			context.PData["controller"] = s
			context.PData["module"] = mod
		}
	}
	if s, err := context.PData.String("controller"); err == nil {
		control = s
	}
	if action, acterr = context.PData.String("action"); acterr == nil {
		act = action
	}
	if act == "" {
		act = "_"
	}

	if ctrl := this.ctrlgroup.Controller(mod, control); ctrl != nil {
		// allows for shortcut action to index
		if acterr == nil && act != "_" && !ctrl.HasAction(act) && ctrl.HasAction("index") {
			act = "_"
			lenupath := len(context.UPath) + 1
			tmpupath := make(web.UPath, lenupath)
			tmpupath[0] = action
			copy(tmpupath[1:lenupath], context.UPath)
			context.UPath = tmpupath
			delete(context.PData, "action")
		}

		cm = controller.NewControlManager(context, ctrl, act)
	}

	return
}
