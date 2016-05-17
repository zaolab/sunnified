package event

type EventTrigger struct {
	eventrouter *EventRouter
	namespace   string
}

func (et *EventTrigger) Namespace() string {
	return et.namespace
}

func (et *EventTrigger) EventRouter() *EventRouter {
	return et.eventrouter
}

func (et *EventTrigger) Fire(name string, info map[string]interface{}) {
	et.eventrouter.route(NewEvent(et.namespace, name, info))
}

func NewEventTrigger(er *EventRouter, namespace string) *EventTrigger {
	return &EventTrigger{
		eventrouter: er,
		namespace:   namespace,
	}
}
