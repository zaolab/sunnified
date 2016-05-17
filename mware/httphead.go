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

func (mw *HTTPHeadMiddleWare) Request(ctxt *web.Context) {
	h := ctxt.Response.Header()

	for name, val := range mw.defaultHeaders {
		for _, v := range val {
			h.Add(name, v)
		}
	}
}

func (mw *HTTPHeadMiddleWare) AddDefaultHeader(name string, value ...string) {
	if arr, exists := mw.defaultHeaders[name]; exists {
		mw.defaultHeaders[name] = append(arr, value...)
	} else {
		newarr := make([]string, len(value))
		copy(newarr, value)
		mw.defaultHeaders[name] = newarr
	}
}
