package mware

import (
	"github.com/zaolab/sunnified/web"
)

func NewHTTPHeadMiddleWare() *HTTPHeadMiddleWare {
	return &HTTPHeadMiddleWare{
		defaultHeaders: make(map[string][]string),
	}
}

func HTTPHeadMiddleWareConstructor() MiddleWare {
	return NewHTTPHeadMiddleWare()
}

type HTTPHeadMiddleWare struct {
	BaseMiddleWare
	defaultHeaders map[string][]string
}

func (this *HTTPHeadMiddleWare) Request(ctxt *web.Context) {
	h := ctxt.Response.Header()

	for name, val := range this.defaultHeaders {
		for _, v := range val {
			h.Add(name, v)
		}
	}
}

func (this *HTTPHeadMiddleWare) AddDefaultHeader(name string, value ...string) {
	if arr, exists := this.defaultHeaders[name]; exists {
		this.defaultHeaders[name] = append(arr, value...)
	} else {
		newarr := make([]string, len(value))
		copy(newarr, value)
		this.defaultHeaders[name] = newarr
	}
}
