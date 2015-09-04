package event

type Event struct {
	namespace string
	name      string
	metas     map[string]interface{}
	info      map[string]interface{}
}

func (this *Event) Id() string {
	return this.namespace + "." + this.name
}

func (this *Event) Namespace() string {
	return this.namespace
}

func (this *Event) Name() string {
	return this.name
}

func (this *Event) MapInfo(varname string, i interface{}) interface{} {
	if val, exists := this.info[varname]; exists {
		i = &val
		return val
	}

	return nil
}

func (this *Event) Info(varname string) (val interface{}) {
	val, _ = this.info[varname]
	return
}

func (this *Event) MapMetaData(varname string, i interface{}) interface{} {
	if val, exists := this.metas[varname]; exists {
		i = &val
		return val
	}

	return nil
}

func (this *Event) MetaData(varname string) (val interface{}) {
	val, _ = this.metas[varname]
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
