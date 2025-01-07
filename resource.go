package main

import (
	"time"
)

type ResourceRenderer interface {
	Render(resource Resource) []string
}

type Resource struct {
	Timestamp time.Time
	Kind      string
	Namespace string
	Name      string
	Object    any
	_extra    map[string]any
	Renderer  ResourceRenderer
}

func NewResource(timestmap time.Time, kind string, namespace string, name string, obj any, renderer ResourceRenderer) Resource {
	return Resource{
		Timestamp: timestmap,
		Kind:      kind,
		Namespace: namespace,
		Name:      name,
		Object:    obj,
		Renderer:  renderer,
		_extra:    map[string]any{},
	}
}

func (r Resource) String() []string {
	if r.Renderer != nil {
		return r.Renderer.Render(r)
	}
	return []string{}
}

func (r Resource) SetExtra(e map[string]any) Resource {
	if e == nil {
		return r
	}

	r._extra = e
	return r
}

func (r Resource) SetExtraKV(k string, v any) Resource {
	r._extra = r.GetExtra()
	r._extra[k] = v
	return r
}
func (r Resource) GetExtra() map[string]any {
	newMap := make(map[string]any)
	for key, value := range r._extra {
		newMap[key] = value
	}

	return newMap
}

func (r Resource) Key() string {
	return r.Kind + "/" + r.Name + "/" + r.Name
}
