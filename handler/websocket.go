package handler

import (
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/web"
)

type WSArgs []string

var (
	argSplit = regexp.MustCompile("\\s+")
	validCmd = regexp.MustCompile("[a-zA-Z][\\w]*")

	argsType = reflect.TypeOf(WSArgs(nil))
	intType  = reflect.TypeOf(0)
	errType  = reflect.TypeOf(error(nil))
	retbType = reflect.TypeOf([]byte(nil))
	retsType = reflect.TypeOf("")
	retmType = reflect.TypeOf(map[string]interface{}(nil))
)

func NewWebSocketHandler(ctrler interface{}, allowcmd bool) *WebSocketHandler {
	ctrlmeta, _, _, _ := controller.MakeControllerMeta(ctrler)

	return &WebSocketHandler{
		ctrl:     ctrlmeta,
		allowcmd: allowcmd,
	}
}

type methMeta struct {
	method  reflect.Value
	rettype int
	isvalid bool
}

type WebSocketHandler struct {
	ctrl     *controller.Meta
	allowcmd bool
}

func (wh *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wh.ServeContextHTTP(web.NewContext(w, r))
}

func (wh *WebSocketHandler) ServeContextHTTP(context *web.Context) {
	var ctrlmgr = controller.NewControlManager(context, wh.ctrl, "_")

	if err := context.ToWebSocket(nil, nil); err != nil {
		context.RaiseAppError("Unable to upgrade to websocket: " + err.Error())
	}

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()

	ctrlmgr.Prepare()

	var (
		ctrler  = ctrlmgr.Controller()
		listen  = &methMeta{method: ctrler.MethodByName("Listen_")}
		methods = make(map[string]*methMeta)
		meth    *methMeta
		ok      bool
		t       reflect.Type
	)

	if listen.method.IsValid() {
		t = listen.method.Type()
		if t.NumIn() != 2 || t.In(0) != intType || t.In(1) != retbType {
			listen.method = reflect.Value{}
		}
	}
	listen.rettype = getRetType(listen.method, t)
	listen.isvalid = listen.method.IsValid()

	for {
		msgT, p, err := context.WebSocket.ReadMessage()

		if err != nil {
			if m := ctrler.MethodByName("Error_"); m.IsValid() && m.Type().NumIn() == 1 &&
				m.Type().In(0) == errType {

				m.Call([]reflect.Value{reflect.ValueOf(err)})
			}

			break
		}

		if wh.allowcmd && msgT == websocket.TextMessage && len(p) > 0 && p[0] == '/' {
			args := WSArgs(argSplit.Split(strings.TrimSpace(string(p[1:len(p)])), -1))
			cmd := strings.Replace(strings.Title(args[0]), "-", "_", -1)

			if cmd[len(cmd)-1] != '_' && validCmd.MatchString(cmd) {
				args = args[1:len(args)]

				if meth, ok = methods[cmd]; !ok {
					m := ctrler.MethodByName(cmd)

					if m.IsValid() {
						t = m.Type()
						if t.NumIn() != 1 || t.In(0) != argsType {
							m = reflect.Value{}
						}
						meth = &methMeta{
							method:  m,
							rettype: getRetType(m, t),
							isvalid: m.IsValid(),
						}
					} else {
						meth = &methMeta{
							method:  m,
							rettype: 0,
							isvalid: false,
						}
					}

					methods[cmd] = meth
				}

				if meth.isvalid {
					write(meth, context.WebSocket, meth.method.Call([]reflect.Value{reflect.ValueOf(args)}))
					continue
				}
			}
		}

		if listen.isvalid {
			write(listen, context.WebSocket,
				listen.method.Call([]reflect.Value{reflect.ValueOf(msgT), reflect.ValueOf(p)}))
		}
	}
}

func getRetType(meth reflect.Value, t reflect.Type) (ret int) {
	if meth.IsValid() {
		if t == nil {
			t = meth.Type()
		}

		if t.NumOut() == 1 {
			ret = 4

			switch t.Out(0) {
			case retbType:
				ret = 1
			case retsType:
				ret = 2
			case retmType:
				ret = 3
			}
		}
	}

	return
}

func write(meth *methMeta, ws *websocket.Conn, res []reflect.Value) {
	switch meth.rettype {
	case 1:
		ws.WriteMessage(websocket.TextMessage, res[0].Bytes())
	case 2:
		ws.WriteMessage(websocket.TextMessage, []byte(res[0].String()))
	case 3, 4:
		ws.WriteJSON(res[0].Interface())
	}
}
