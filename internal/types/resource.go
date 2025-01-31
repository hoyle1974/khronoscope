package types

import "time"

type Marked interface {
	GetMark() string
}

// Resource represents the minimal interface needed to display resources
type Resource interface {
	GetUID() string
	GetKind() string
	GetNamespace() string
	GetName() string
	GetTimestamp() time.Time
	GetDetails() []string
	String() string
}
