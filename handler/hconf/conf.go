package hconf

import (
	"github.com/zaolab/sunnified/handler"
	"net/http"
)

const (
	DEFAULT_MODULE     string = "forum"
	DEFAULT_CONTROLLER string = "default"
	DEFAULT_ACTION     string = "index"
)

var HANDLERS map[string]interface{} = map[string]func() http.Handler{
	"DynamicHandler": handler.NewDynamicHandler,
}
