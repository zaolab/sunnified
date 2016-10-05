package event

type Trigger struct {
	eventrouter *Router
	namespace   string
}

func (et *Trigger) Namespace() string {
	return et.namespace
}

func (et *Trigger) EventRouter() *Router {
	return et.eventrouter
}

func (et *Trigger) Fire(name string, info map[string]interface{}) {
	et.eventrouter.route(NewEvent(et.namespace, name, info))
}

func NewEventTrigger(er *Router, namespace string) *Trigger {
	return &Trigger{
		eventrouter: er,
		namespace:   namespace,
	}
}
