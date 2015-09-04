package mware

/*
import (
	"net/http"
	"reflect"
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/web"
	"sync"
)

type PluginMiddleWare struct {
	BaseMiddleWare
	reqvalue reflect.Value
}

func (this *PluginMiddleWare) Request(ctxt *web.Context) {
	this.BaseMiddleWare.Context = ctxt
	wait := sync.WaitGroup{}
	plugins := make([]reflect.Value, 0)
	this.reqvalue = reflect.ValueOf(*ctxt.Request)
	args := []reflect.Value{this.reqvalue}

	for _, plugin := range plugins {
		wait.Add(1)
		go func() {
			defer wait.Done()
			plugin.Call(args)
		}()
	}

	wait.Wait()
}

func (this *PluginMiddleWare) Controller(con *controller.ControlManager) {
	wait := sync.WaitGroup{}
	plugins := make([]reflect.Value, 0)
	args := []reflect.Value{this.reqvalue, reflect.ValueOf(con)}

	for _, plugin := range plugins {
		wait.Add(1)
		go func() {
			defer wait.Done()
			plugin.Call(args)
		}()
	}

	wait.Wait()
}

func (this *PluginMiddleWare) View(view mvc.View) {
	wait := sync.WaitGroup{}
	plugins := make([]reflect.Value, 0)
	args := []reflect.Value{this.reqvalue, reflect.ValueOf(view)}

	for _, plugin := range plugins {
		wait.Add(1)
		go func() {
			defer wait.Done()
			plugin.Call(args)
		}()
	}

	wait.Wait()
}

func (this *PluginMiddleWare) Response() {
	wait := sync.WaitGroup{}
	plugins := make([]reflect.Value, 0)
	header := this.BaseMiddleWare.Context.Response.Header()
	copyheader := make(map[string][]string)
	for k, v := range header {
		val := make([]string, len(v))
		copy(val, v)
		copyheader[k] = val
	}
	args := []reflect.Value{this.reqvalue, reflect.ValueOf(http.Header(copyheader))}

	for _, plugin := range plugins {
		wait.Add(1)
		go func() {
			defer wait.Done()
			plugin.Call(args)
		}()
	}

	wait.Wait()
}
*/
