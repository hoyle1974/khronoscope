package dao

import (
	"time"

	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/hoyle1974/khronoscope/internal/temporal"
)

type KhronoStore interface {
	GetResourcesAt(timestamp time.Time, kind string, namespace string) []resources.Resource
	GetResourceAt(timestamp time.Time, uid string) (resources.Resource, error)
	GetTimeRange() (time.Time, time.Time)
	AddResource(resource resources.Resource)
	UpdateResource(resource resources.Resource)
	DeleteResource(resource resources.Resource)
	SetLabel(time.Time, string)
	GetLabel(time time.Time) string
	GetNextLabelTime(time.Time) time.Time
	GetPrevLabelTime(time.Time) time.Time
	Save(string)
	Size() int
}

func New() KhronoStore {
	return &inMemoryImpl{
		meta:      temporal.New(),
		resources: temporal.New(),
	}
}

func NewPersist() KhronoStore {
	return &persistImpl{
		resources: temporal.NewDiskMap(),
	}
}
