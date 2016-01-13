package controller

import (
	"github.com/gorilla/websocket"
	"github.com/zaolab/sunnified/web"
	"sync"
)

func NewWebSocketChat() *WebsocketChat {
	return &WebsocketChat{
		channels: make(map[string]map[string]map[*web.Context]*websocket.Conn),
	}
}

type WebsocketChat struct {
	channels map[string]map[string]map[*web.Context]*websocket.Conn
	mutex    sync.RWMutex
}

func (this *WebsocketChat) AddChannel(channel string) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.channels[channel] == nil {
		this.channels[channel] = make(map[string]map[*web.Context]*websocket.Conn)
	}
}

func (this *WebsocketChat) AddClient(channel, uid string, context *web.Context) {
	if uid == "" {
		uid = getUserId(context.Session)
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.channels[channel] != nil {
		if this.channels[channel][uid] == nil {
			this.channels[channel][uid] = make(map[*web.Context]*websocket.Conn)
		}
		this.channels[channel][uid][context] = context.WebSocket
	}
}

func (this *WebsocketChat) RemoveClient(channel, uid string, context *web.Context) {
	if uid == "" {
		uid = getUserId(context.Session)
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	if context == nil {
		delete(this.channels[channel], uid)
	} else if this.channels[channel] != nil {
		delete(this.channels[channel][uid], context)
		if len(this.channels[channel][uid]) == 0 {
			delete(this.channels[channel], uid)
		}
	}
}

func (this *WebsocketChat) Client(channel, uid string) map[*web.Context]*websocket.Conn {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	if this.channels[channel] != nil && this.channels[channel][uid] != nil{
		m := make(map[*web.Context]*websocket.Conn)

		for k, v := range this.channels[channel][uid] {
			m[k] = v
		}

		return m
	}

	return nil
}

func (this *WebsocketChat) MessageClient(channel, uid string, msgT int, msg []byte) bool {
	return this.message_(channel, uid, msgT, msg)
}

func (this *WebsocketChat) MessageJSONClient(channel, uid string, msg interface{}) bool {
	return this.message_(channel, uid, -1, msg)
}

func (this *WebsocketChat) Broadcast(channel string, msgT int, msg []byte) {
	this.broadcast_(channel, msgT, msg)
}

func (this *WebsocketChat) BroadcastJSON(channel string, msg interface{}) {
	this.broadcast_(channel, -1, msg)
}

func (this *WebsocketChat) message_(channel, uid string, msgT int, msg interface{}) bool {
	client := this.Client(channel, uid)

	if client != nil && len(client) > 0 {
		var (
			removed = 0
			err error
		)

		for ctxt, ws := range client {
			if msgT == -1 {
				err = ws.WriteJSON(msg)
			} else {
				err = ws.WriteMessage(msgT, msg.([]byte))
			}

			if err != nil {
				this.RemoveClient(channel, uid, ctxt)
				removed++
			}
		}

		return removed < len(client)
	}

	return false
}

func (this *WebsocketChat) broadcast_(channel string, msgT int, msg interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	if cc, ok := this.channels[channel]; ok {
		queue := make(chan struct{}, 500) // max 500 goroutines concurrent
		wg := sync.WaitGroup{}
		wg.Add(len(cc))

		for uid, ctxtarr := range cc {
			for ctxt, ws := range ctxtarr {
				queue <- struct{}{}

				go func(uid string, ctxt *web.Context, ws *websocket.Conn) {
					defer func() {
						<-queue
						wg.Done()
					}()

					var err error

					if msgT == -1 {
						err = ws.WriteJSON(msg)
					} else {
						err = ws.WriteMessage(msgT, msg.([]byte))
					}

					if err != nil {
						// async this so we do not have a deadlock...
						// all remove clients will only complete after the whole channel is looped
						go this.RemoveClient(channel, uid, ctxt)
					}
				}(uid, ctxt, ws)
			}
		}

		wg.Wait()
		close(queue)
	}
}

func getUserId(sess web.SessionManager) string {
	if sess == nil {
		return ""
	}
	if au := sess.AuthUser(); au != nil {
		return au.ID()
	}

	return ""
}
