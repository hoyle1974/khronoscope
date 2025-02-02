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
