package controller

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/zaolab/sunnified/web"
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

func (wc *WebsocketChat) AddChannel(channel string) {
	wc.mutex.Lock()
	defer wc.mutex.Unlock()

	if wc.channels[channel] == nil {
		wc.channels[channel] = make(map[string]map[*web.Context]*websocket.Conn)
	}
}

func (wc *WebsocketChat) AddClient(channel, uid string, context *web.Context) {
	if uid == "" {
		uid = getUserID(context.Session)
	}

	wc.mutex.Lock()
	defer wc.mutex.Unlock()

	if wc.channels[channel] != nil {
		if wc.channels[channel][uid] == nil {
			wc.channels[channel][uid] = make(map[*web.Context]*websocket.Conn)
		}
		wc.channels[channel][uid][context] = context.WebSocket
	}
}

func (wc *WebsocketChat) RemoveClient(channel, uid string, context *web.Context) {
	if uid == "" {
		uid = getUserID(context.Session)
	}

	wc.mutex.Lock()
	defer wc.mutex.Unlock()

	if context == nil {
		delete(wc.channels[channel], uid)
	} else if wc.channels[channel] != nil {
		delete(wc.channels[channel][uid], context)
		if len(wc.channels[channel][uid]) == 0 {
			delete(wc.channels[channel], uid)
		}
	}
}

func (wc *WebsocketChat) Client(channel, uid string) map[*web.Context]*websocket.Conn {
	wc.mutex.RLock()
	defer wc.mutex.RUnlock()

	if wc.channels[channel] != nil && wc.channels[channel][uid] != nil {
		m := make(map[*web.Context]*websocket.Conn)

		for k, v := range wc.channels[channel][uid] {
			m[k] = v
		}

		return m
	}

	return nil
}

func (wc *WebsocketChat) MessageClient(channel, uid string, msgT int, msg []byte) bool {
	return wc.message_(channel, uid, msgT, msg)
}

func (wc *WebsocketChat) MessageJSONClient(channel, uid string, msg interface{}) bool {
	return wc.message_(channel, uid, -1, msg)
}

func (wc *WebsocketChat) Broadcast(channel string, msgT int, msg []byte) {
	wc.broadcast_(channel, msgT, msg)
}

func (wc *WebsocketChat) BroadcastJSON(channel string, msg interface{}) {
	wc.broadcast_(channel, -1, msg)
}

func (wc *WebsocketChat) message_(channel, uid string, msgT int, msg interface{}) bool {
	client := wc.Client(channel, uid)

	if client != nil && len(client) > 0 {
		var (
			removed = 0
			err     error
		)

		for ctxt, ws := range client {
			if msgT == -1 {
				err = ws.WriteJSON(msg)
			} else {
				err = ws.WriteMessage(msgT, msg.([]byte))
			}

			if err != nil {
				wc.RemoveClient(channel, uid, ctxt)
				removed++
			}
		}

		return removed < len(client)
	}

	return false
}

func (wc *WebsocketChat) broadcast_(channel string, msgT int, msg interface{}) {
	wc.mutex.RLock()
	defer wc.mutex.RUnlock()

	if cc, ok := wc.channels[channel]; ok {
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
						go wc.RemoveClient(channel, uid, ctxt)
					}
				}(uid, ctxt, ws)
			}
		}

		wg.Wait()
		close(queue)
	}
}

func getUserID(sess web.SessionManager) string {
	if sess == nil {
		return ""
	}
	if au := sess.AuthUser(); au != nil {
		return au.ID()
	}

	return ""
}
