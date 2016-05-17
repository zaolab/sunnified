package handler

import (
	"net/http"
	"strings"

	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/router"
	"github.com/zaolab/sunnified/web"
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

func (ch *ControllerHandler) SetController(mod, ctrler string) {
	ch.controlmeta = controller.Controller(mod, ctrler)
}

func (ch *ControllerHandler) SetControllerMeta(controlmeta *controller.ControllerMeta) {
	ch.controlmeta = controlmeta
}

func (ch *ControllerHandler) SetAction(action string) {
	ch.action = action
}

func (ch *ControllerHandler) ServeOptions(w http.ResponseWriter, r *http.Request, origin map[string]string) {
	ch.ServeContextOptions(web.NewContext(w, r), origin)
}

func (ch *ControllerHandler) ServeContextOptions(ctxt *web.Context, origin map[string]string) {
	router.SetHeaderOrigin(ctxt.Response, ctxt.Request, origin)
	header := ctxt.Response.Header()
	ctrlmgr := ch.GetControlManager(ctxt)
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

func (ch *ControllerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ch.ServeContextHTTP(web.NewContext(w, r))
}

func (ch *ControllerHandler) ServeContextHTTP(ctxt *web.Context) {
	if ctrlmgr := ch.GetControlManager(ctxt); ctrlmgr != nil {
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

func (ch *ControllerHandler) GetControlManager(context *web.Context) (cm *controller.ControlManager) {
	var (
		act    string = ch.action
		action string
		acterr error
	)

	if action, acterr = context.PData.String("action"); acterr == nil {
		act = action
	}
	if act == "" {
		act = "_"
	}

	if acterr == nil && act != "_" && !ch.controlmeta.HasAction(act) && ch.controlmeta.HasAction("index") {
		act = "_"
		lenupath := len(context.UPath) + 1
		tmpupath := make(web.UPath, lenupath)
		tmpupath[0] = action
		copy(tmpupath[1:lenupath], context.UPath)
		context.UPath = tmpupath
		delete(context.PData, "action")
	}

	cm = controller.NewControlManager(context, ch.controlmeta, act)

	return
}
