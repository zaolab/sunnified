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

func (this *EventRouter) CreateTrigger(namespace string) *EventTrigger {
	return NewEventTrigger(this, namespace)
}

func (this *EventRouter) IsSubRouter() bool {
	return this.base != nil
}

func (this *EventRouter) SubRouter() *EventRouter {
	return this.base
}

func (this *EventRouter) route(event *Event) {
	if event != nil && this.metas != nil {
		if event.metas == nil {
			event.metas = make(M)
		}

		for k, v := range this.metas {
			if _, exists := event.metas[k]; !exists {
				event.metas[k] = v
			}
		}

		this.mutex.RLock()
		defer this.mutex.RUnlock()
		if lis, exists := this.listeners[JoinId(event.namespace, event.name)]; exists {
			for _, f := range lis {
				f(event)
			}
		}

		if this.base != nil {
			this.base.route(event)
		}
	}
}

func (this *EventRouter) Listen(id string, f Listener) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if _, exists := this.listeners[id]; !exists {
		this.listeners[id] = make([]Listener, 0, 5)
	}

	this.listeners[id] = append(this.listeners[id], f)
}

func (this *EventRouter) NewSubRouter(metas M) *EventRouter {
	return &EventRouter{
		metas: metas,
		base:  this,
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
