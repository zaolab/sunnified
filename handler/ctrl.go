package handler

import (
	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/router"
	"github.com/zaolab/sunnified/web"
	"net/http"
	"strings"
)

type ControllerHandler struct {
	controlmeta *controller.ControllerMeta
	action      string
}

func NewControllerHandler() http.Handler {
	return &ControllerHandler{}
}

func NewDefaultControllerHandler(controlmeta *controller.ControllerMeta, action string) *ControllerHandler {
	return &ControllerHandler{
		controlmeta: controlmeta,
		action:      action,
	}
}

func NewNamedControllerHandler(mod, ctrler, action string) (chand *ControllerHandler) {
	chand = &ControllerHandler{action: action}
	chand.SetController(mod, ctrler)
	return
}

func (this *ControllerHandler) SetController(mod, ctrler string) {
	this.controlmeta = controller.Controller(mod, ctrler)
}

func (this *ControllerHandler) SetControllerMeta(controlmeta *controller.ControllerMeta) {
	this.controlmeta = controlmeta
}

func (this *ControllerHandler) SetAction(action string) {
	this.action = action
}

func (this *ControllerHandler) ServeOptions(w http.ResponseWriter, r *http.Request, origin map[string]string) {
	this.ServeContextOptions(web.NewContext(w, r), origin)
}

func (this *ControllerHandler) ServeContextOptions(ctxt *web.Context, origin map[string]string) {
	router.SetHeaderOrigin(ctxt.Response, ctxt.Request, origin)
	header := ctxt.Response.Header()
	ctrlmgr := this.GetControlManager(ctxt)
	mlist := ctrlmgr.AvailableMethodsList()
	var allow string = "HEAD, OPTIONS"
	mliststr := strings.Join(mlist, ", ")
	if mliststr != "" {
		allow += ", " + mliststr
	}
	header.Set("Allow", allow)
	header.Set("Access-Control-Allow-Methods", mliststr)
	ctxt.Response.WriteHeader(200)
}

func (this *ControllerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	this.ServeContextHTTP(web.NewContext(w, r))
}

func (this *ControllerHandler) ServeContextHTTP(ctxt *web.Context) {
	if ctrlmgr := this.GetControlManager(ctxt); ctrlmgr != nil {
		ctrlmgr.Prepare()
		state, vw := ctrlmgr.Execute()
		if state != -1 && vw == nil {
			ErrorHtml(ctxt.Response, ctxt.Request, state)
			return
		}
		ctrlmgr.PublishView()
		ctrlmgr.Cleanup()
	} else {
		NotFound(ctxt.Response, ctxt.Request)
	}
}

func (this *ControllerHandler) GetControlManager(context *web.Context) (cm *controller.ControlManager) {
	var (
		act    string = this.action
		action string
		acterr error
	)

	if action, acterr = context.PData.String("action"); acterr == nil {
		act = action
	}
	if act == "" {
		act = "_"
	}

	if acterr == nil && act != "_" && !this.controlmeta.HasAction(act) && this.controlmeta.HasAction("index") {
		act = "_"
		lenupath := len(context.UPath) + 1
		tmpupath := make(web.UPath, lenupath)
		tmpupath[0] = action
		copy(tmpupath[1:lenupath], context.UPath)
		context.UPath = tmpupath
		delete(context.PData, "action")
	}

	cm = controller.NewControlManager(context, this.controlmeta, act)

	return
}
