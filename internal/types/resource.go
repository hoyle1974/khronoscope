package types

import "time"

// Resource represents the minimal interface needed to display resources
type Resource interface {
	GetUID() string
	GetKind() string
	GetNamespace() string
	GetName() string
	GetTimestamp() time.Time
	GetDetails() []string
	String() string
	GetExtra() any
}

func NewPendingResource(r Resource) Resource {
	return pendingResource{r}
}

type pendingResource struct {
	r Resource
}

func (p pendingResource) GetUID() string          { return p.r.GetUID() }
func (p pendingResource) GetKind() string         { return p.r.GetKind() }
func (p pendingResource) GetNamespace() string    { return p.r.GetNamespace() }
func (p pendingResource) GetName() string         { return "pending . . ." }
func (p pendingResource) GetTimestamp() time.Time { return p.r.GetTimestamp() }
func (p pendingResource) GetDetails() []string    { return []string{"pending . . . "} }
func (p pendingResource) String() string          { return "pending . . ." }
func (p pendingResource) GetExtra() any           { return nil }
