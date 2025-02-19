package resources

import (
	"strings"
	"sync"
	"time"

	"github.com/hoyle1974/khronoscope/internal/serializable"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A Resource represents a k8s resource like a Pod, ReplicaSet, or Node.

type ResourceRenderer interface {
	Render(resource Resource, detailed bool) []string
}

var renderMapLock sync.RWMutex
var renderMap = map[string]ResourceRenderer{}

func RegisterResourceRenderer(k string, r ResourceRenderer) {
	renderMapLock.Lock()
	defer renderMapLock.Unlock()
	renderMap[k] = r
}
func GetRenderer(k string) ResourceRenderer {
	renderMapLock.RLock()
	defer renderMapLock.RUnlock()
	return renderMap[k]
}

type K8sResource interface {
	GetObjectMeta() v1.Object
}

type Copyable interface {
	Copy() Copyable
}

type Resource struct {
	Uid       string            // The Uid of the k8s object
	Timestamp serializable.Time // The timestamp that this resource is valid for
	Kind      string            // The k8s kind of resource
	Namespace string            // The k8s namespace, may be empty for things like namespace and node resources
	Name      string            // The name of the resource
	Extra     Copyable          // This should be a custom, gob registered and serializable object if used
}

func (r Resource) GetUID() string          { return r.Uid }
func (r Resource) GetKind() string         { return r.Kind }
func (r Resource) GetNamespace() string    { return r.Namespace }
func (r Resource) GetName() string         { return r.Name }
func (r Resource) GetTimestamp() time.Time { return r.Timestamp.Time }
func (r Resource) GetExtra() any           { return r.Extra }

func NewK8sResource(kind string, obj K8sResource, extra Copyable) Resource {
	r := Resource{
		Uid:       string(obj.GetObjectMeta().GetUID()),
		Timestamp: serializable.Time{Time: time.Now()},
		Kind:      kind,
		Namespace: obj.GetObjectMeta().GetNamespace(),
		Name:      obj.GetObjectMeta().GetName(),
		Extra:     extra,
	}

	return r
}

// Render the details of the Resource
func (r Resource) GetDetails() []string {
	rr := GetRenderer(r.Kind)
	if rr == nil {
		return []string{}
	}
	return rr.Render(r, true)
}

// Render a short version of the Resource
func (r Resource) String() string {
	rr := GetRenderer(r.Kind)
	if rr == nil {
		return r.Key()
	}
	return strings.Join(rr.Render(r, false), " ")
}

func NewResource(uuid string, timestmap time.Time, kind string, namespace string, name string) Resource {
	return Resource{
		Uid:       uuid,
		Timestamp: serializable.Time{Time: timestmap},
		Kind:      kind,
		Namespace: namespace,
		Name:      name,
	}
}

func (r Resource) Key() string {
	//return r.Kind + "/" + r.Namespace + "/" + r.Name
	return r.Uid
}
