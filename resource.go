package main

import "time"

type Resource struct {
	Timestamp time.Time
	Kind      string
	Namespace string
	Name      string
	Object    any
	_extra    map[string]any
}

func NewResource(timestmap time.Time, kind string, namespace string, name string, obj any) Resource {
	return Resource{
		Timestamp: timestmap,
		Kind:      kind,
		Namespace: namespace,
		Name:      name,
		Object:    obj,
		_extra:    map[string]any{},
	}
}

func (r Resource) SetExtra(e map[string]any) Resource {
	if e == nil {
		panic("Can not set extra to a nil map")
	}
	r._extra = e
	return r
}
func (r Resource) SetExtraKV(k string, v any) Resource {
	if r._extra == nil {
		r._extra = map[string]any{}
	}
	r._extra[k] = v
	return r
}
func (r Resource) GetExtra() map[string]any {
	return r._extra
}

func (r Resource) Key() string {
	return r.Kind + "/" + r.Name + "/" + r.Name
}
