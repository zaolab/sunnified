package event

import (
	"strings"
	"sync"
)

type Listener func(*Event)

type M map[string]interface{}

type Router struct {
	base      *Router
	mutex     sync.RWMutex
	metas     M
	listeners map[string][]Listener
}

func (er *Router) CreateTrigger(namespace string) *Trigger {
	return NewEventTrigger(er, namespace)
}

func (er *Router) IsSubRouter() bool {
	return er.base != nil
}

func (er *Router) SubRouter() *Router {
	return er.base
}

func (er *Router) route(event *Event) {
	if event != nil && er.metas != nil {
		if event.metas == nil {
			event.metas = make(M)
		}

		for k, v := range er.metas {
			if _, exists := event.metas[k]; !exists {
				event.metas[k] = v
			}
		}

		er.mutex.RLock()
		defer er.mutex.RUnlock()
		if lis, exists := er.listeners[JoinID(event.namespace, event.name)]; exists {
			for _, f := range lis {
				f(event)
			}
		}

		if er.base != nil {
			er.base.route(event)
		}
	}
}

func (er *Router) Listen(id string, f Listener) {
	er.mutex.Lock()
	defer er.mutex.Unlock()
	if _, exists := er.listeners[id]; !exists {
		er.listeners[id] = make([]Listener, 0, 5)
	}

	er.listeners[id] = append(er.listeners[id], f)
}

func (er *Router) NewSubRouter(metas M) *Router {
	return &Router{
		metas: metas,
		base:  er,
	}
}

func NewEventRouter(metas M) *Router {
	return &Router{
		metas:     metas,
		listeners: make(map[string][]Listener),
	}
}

func SplitID(id string) (string, string) {
	idsplit := strings.SplitN(id, ".", 2)
	return idsplit[0], idsplit[1]
}

func JoinID(namespace, name string) string {
	return namespace + "." + name
}
