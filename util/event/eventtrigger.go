package event

type EventTrigger struct {
	eventrouter *EventRouter
	namespace   string
}

func (this *EventTrigger) Namespace() string {
	return this.namespace
}

func (this *EventTrigger) EventRouter() *EventRouter {
	return this.eventrouter
}

func (this *EventTrigger) Fire(name string, info map[string]interface{}) {
	this.eventrouter.route(NewEvent(this.namespace, name, info))
}

func NewEventTrigger(er *EventRouter, namespace string) *EventTrigger {
	return &EventTrigger{
		eventrouter: er,
		namespace:   namespace,
	}
}
