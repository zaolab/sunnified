package event

import (
	"strings"
	"sync"
)

type Listener func(*Event)

type M map[string]interface{}

type EventRouter struct {
	base      *EventRouter
	mutex     sync.RWMutex
	metas     M
	listeners map[string][]Listener
}

func (er *EventRouter) CreateTrigger(namespace string) *EventTrigger {
	return NewEventTrigger(er, namespace)
}

func (er *EventRouter) IsSubRouter() bool {
	return er.base != nil
}

func (er *EventRouter) SubRouter() *EventRouter {
	return er.base
}

func (er *EventRouter) route(event *Event) {
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
		if lis, exists := er.listeners[JoinId(event.namespace, event.name)]; exists {
			for _, f := range lis {
				f(event)
			}
		}

		if er.base != nil {
			er.base.route(event)
		}
	}
}

func (er *EventRouter) Listen(id string, f Listener) {
	er.mutex.Lock()
	defer er.mutex.Unlock()
	if _, exists := er.listeners[id]; !exists {
		er.listeners[id] = make([]Listener, 0, 5)
	}

	er.listeners[id] = append(er.listeners[id], f)
}

func (er *EventRouter) NewSubRouter(metas M) *EventRouter {
	return &EventRouter{
		metas: metas,
		base:  er,
	}
}

func NewEventRouter(metas M) *EventRouter {
	return &EventRouter{
		metas:     metas,
		listeners: make(map[string][]Listener),
	}
}

func SplitId(id string) (string, string) {
	idsplit := strings.SplitN(id, ".", 2)
	return idsplit[0], idsplit[1]
}

func JoinId(namespace, name string) string {
	return namespace + "." + name
}
