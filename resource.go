package main

import (
	"strings"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A Resource represents a k8s resource like a Pod, ReplicaSet, or Node.

type ResourceRenderer interface {
	Render(resource Resource, detailed bool) []string
}

type K8sResource interface {
	GetObjectMeta() v1.Object
}

type Resource struct {
	Uid       string           // The Uid of the k8s object
	Timestamp time.Time        // The timestamp that this resource is valid for
	Kind      string           // The k8s kind of resource
	Namespace string           // The k8s namespace, may be empty for things like namespace and node resources
	Name      string           // The name of the resource
	Object    any              // The actual k8s object, often used by renderers
	_extra    map[string]any   // Any extra data, like metrics, attached to this resource
	Renderer  ResourceRenderer // The renderer for this resource
}

func NewK8sResource(kind string, obj K8sResource, renderer ResourceRenderer) Resource {
	return Resource{
		Uid:       string(obj.GetObjectMeta().GetUID()),
		Timestamp: time.Now(),
		Kind:      kind,
		Namespace: obj.GetObjectMeta().GetNamespace(),
		Name:      obj.GetObjectMeta().GetName(),
		Object:    obj,
		Renderer:  renderer,
		_extra:    map[string]any{},
	}
}

func NewResource(uuid string, timestmap time.Time, kind string, namespace string, name string, obj any, renderer ResourceRenderer) Resource {
	return Resource{
		Uid:       uuid,
		Timestamp: timestmap,
		Kind:      kind,
		Namespace: namespace,
		Name:      name,
		Object:    obj,
		Renderer:  renderer,
		_extra:    map[string]any{},
	}
}

func (r Resource) String() string {
	if r.Renderer != nil {
		return strings.Join(r.Renderer.Render(r, false), " ")
	}
	return r.Key()
}

func (r Resource) Details() []string {
	if r.Renderer != nil {
		return r.Renderer.Render(r, true)
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
	return r.Kind + "/" + r.Namespace + "/" + r.Name
}
