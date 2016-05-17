package handler

import (
	"net/http"
	"strings"
	"sync"

	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/router"
	"github.com/zaolab/sunnified/web"
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

func (dh *DynamicHandler) SetModule(mod string) {
	dh.mutex.Lock()
	defer dh.mutex.Unlock()
	dh.module = mod
}

func (dh *DynamicHandler) SetController(control string) {
	dh.mutex.Lock()
	defer dh.mutex.Unlock()
	dh.controller = control
}

func (dh *DynamicHandler) SetAction(action string) {
	dh.mutex.Lock()
	defer dh.mutex.Unlock()
	dh.action = action
}

func (dh *DynamicHandler) Module() string {
	dh.mutex.RLock()
	defer dh.mutex.RUnlock()
	return dh.module
}

func (dh *DynamicHandler) Controller() string {
	dh.mutex.RLock()
	defer dh.mutex.RUnlock()
	return dh.controller
}

func (dh *DynamicHandler) Action() string {
	dh.mutex.RLock()
	defer dh.mutex.RUnlock()
	return dh.action
}

func (dh *DynamicHandler) ServeOptions(w http.ResponseWriter, r *http.Request, origin map[string]string) {
	dh.ServeContextOptions(web.NewContext(w, r), origin)
}

func (dh *DynamicHandler) ServeContextOptions(context *web.Context, origin map[string]string) {
	router.SetHeaderOrigin(context.Response, context.Request, origin)

	var (
		header  http.Header                = context.Response.Header()
		ctrlmgr *controller.ControlManager = dh.GetControlManager(context)
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

func (dh *DynamicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dh.ServeContextHTTP(web.NewContext(w, r))
}

func (dh *DynamicHandler) ServeContextHTTP(context *web.Context) {
	if ctrlmgr := dh.GetControlManager(context); ctrlmgr != nil {
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

func (dh *DynamicHandler) GetControlManager(context *web.Context) (cm *controller.ControlManager) {
	dh.mutex.RLock()
	var mod, control, act string = dh.module, dh.controller, dh.action
	dh.mutex.RUnlock()

	var acterr error
	var action string

	// if mod is not found, the default mod will be used
	// this allows user to use shortcut path
	// especially when there is only one mod
	if s, err := context.PData.String("module"); err == nil {
		if dh.ctrlgroup.HasModule(s) || !dh.ctrlgroup.HasController(mod, s) {
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

	if ctrl := dh.ctrlgroup.Controller(mod, control); ctrl != nil {
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
