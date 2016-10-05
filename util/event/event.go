package event

type Event struct {
	namespace string
	name      string
	metas     map[string]interface{}
	info      map[string]interface{}
}

func (e *Event) ID() string {
	return e.namespace + "." + e.name
}

func (e *Event) Namespace() string {
	return e.namespace
}

func (e *Event) Name() string {
	return e.name
}

func (e *Event) MapInfo(varname string, i interface{}) interface{} {
	if val, exists := e.info[varname]; exists {
		i = &val
		return val
	}

	return nil
}

func (e *Event) Info(varname string) (val interface{}) {
	val, _ = e.info[varname]
	return
}

func (e *Event) MapMetaData(varname string, i interface{}) interface{} {
	if val, exists := e.metas[varname]; exists {
		i = &val
		return val
	}

	return nil
}

func (e *Event) MetaData(varname string) (val interface{}) {
	val, _ = e.metas[varname]
	return
}

func NewEvent(nspace, name string, info map[string]interface{}) (e *Event) {
	e = &Event{
		namespace: nspace,
		name:      name,
		info:      info,
	}
	return
}
