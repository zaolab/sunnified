package mware

import (
	"errors"
	"sync"

	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/web"
)

type MiddleWareConstructor func() MiddleWare

var (
	mutex                   = sync.RWMutex{}
	middlewares             = make(map[string]MiddleWareConstructor)
	ErrMiddleWareNameExists = errors.New("Name given for middleware already exists")
)

func AddMiddleWare(name string, f MiddleWareConstructor) (err error) {
	mutex.Lock()
	defer mutex.Unlock()

	if _, exists := middlewares[name]; !exists {
		middlewares[name] = f
	} else {
		err = ErrMiddleWareNameExists
	}

	return
}

func GetMiddleWare(name string) MiddleWare {
	mutex.Lock()
	defer mutex.RUnlock()

	if mwc, exists := middlewares[name]; exists {
		return mwc()
	}

	return nil
}

type MiddleWare interface {
	Request(*web.Context)
	Body(*web.Context)
	Controller(*web.Context, *controller.ControlManager)
	View(*web.Context, mvc.View)
	Response(*web.Context)
	Cleanup(*web.Context)
}

type BaseMiddleWare struct {
}

func (mw BaseMiddleWare) Request(ctxt *web.Context) {
}

func (mw BaseMiddleWare) Body(ctxt *web.Context) {
}

func (mw BaseMiddleWare) Controller(ctxt *web.Context, cm *controller.ControlManager) {
}

func (mw BaseMiddleWare) View(ctxt *web.Context, vw mvc.View) {
}

func (mw BaseMiddleWare) Response(ctxt *web.Context) {
}

func (mw BaseMiddleWare) Cleanup(ctxt *web.Context) {
}
