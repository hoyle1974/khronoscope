package dao

import (
	"time"

	"github.com/hoyle1974/khronoscope/internal/resources"

	temp "github.com/hoyle1974/temporal"
	"github.com/hoyle1974/temporal/storage"
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

func NewStore(storageManager storage.System) (KhronoStore, error) {
	m, err := temp.NewMap(storageManager)
	if err != nil {
		return nil, err
	}
	s := &storeImpl{
		storage: m,
	}

	return s, nil
}
