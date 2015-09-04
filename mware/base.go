package mware

import (
	"errors"
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/web"
	"sync"
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

func (this BaseMiddleWare) Request(ctxt *web.Context) {
}

func (this BaseMiddleWare) Body(ctxt *web.Context) {
}

func (this BaseMiddleWare) Controller(ctxt *web.Context, cm *controller.ControlManager) {
}

func (this BaseMiddleWare) View(ctxt *web.Context, vw mvc.View) {
}

func (this BaseMiddleWare) Response(ctxt *web.Context) {
}

func (this BaseMiddleWare) Cleanup(ctxt *web.Context) {
}
