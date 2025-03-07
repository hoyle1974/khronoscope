package dao

import (
	"errors"
	"sync"
	"time"

	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/hoyle1974/khronoscope/internal/temporal"
	"github.com/rs/zerolog/log"
)

type persistImpl struct {
	lock      sync.Mutex
	resources *temporal.DiskMapImpl
}

func (d *persistImpl) Size() int {
	return 0
}

func (d *persistImpl) Save(filename string) {
}

func (d *persistImpl) SetLabel(time time.Time, label string) {
	d.resources.SetMeta(time, []byte(label))
}

func (d *persistImpl) GetLabel(time time.Time) string {
	currentLabel := ""
	if temp, ok := d.resources.GetMeta(time); ok {
		currentLabel = string(temp)
	}
	return currentLabel
}

func (d *persistImpl) GetNextLabelTime(time time.Time) time.Time {
	if t, err := d.resources.FindMetaTime(time, 1); err == nil {
		return t
	}
	_, next := d.GetTimeRange()
	return next
}

func (d *persistImpl) GetPrevLabelTime(time time.Time) time.Time {
	if t, err := d.resources.FindMetaTime(time, -1); err == nil {
		return t
	}
	prev, _ := d.GetTimeRange()
	return prev
}

func (d *persistImpl) GetTimeRange() (time.Time, time.Time) {
	return d.resources.GetTimeRange()
}

func (d *persistImpl) AddResource(resource resources.Resource) {
	log.Debug().Str("Resource", resource.Name).Str("Uid", resource.Uid).Msg("Add")

	d.lock.Lock()
	defer d.lock.Unlock()

	d.resources.Update(resource.Timestamp.Time, resource.Key(), resources.ResourceToBytes(resource))
}

func (d *persistImpl) UpdateResource(resource resources.Resource) {
	log.Debug().Str("Resource", resource.Name).Str("Uid", resource.Uid).Msg("Update")

	d.lock.Lock()
	defer d.lock.Unlock()

	d.resources.Update(resource.Timestamp.Time, resource.Key(), resources.ResourceToBytes(resource))

}

func (d *persistImpl) DeleteResource(resource resources.Resource) {
	log.Debug().Str("Resource", resource.Name).Str("Uid", resource.Uid).Msg("Delete")
	d.lock.Lock()
	defer d.lock.Unlock()

	d.resources.Remove(resource.Timestamp.Time, resource.Key())
}

func (d *persistImpl) GetResourceAt(timestamp time.Time, uid string) (resources.Resource, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	r, ok := d.resources.GetStateAtTime(timestamp)[uid]
	if !ok {
		return resources.Resource{}, errors.New("no resource found")
	}

	return resources.BytesToResource(r), nil
}

func (d *persistImpl) GetResourcesAt(timestamp time.Time, kind string, namespace string) []resources.Resource {
	d.lock.Lock()
	defer d.lock.Unlock()
	m := d.resources.GetStateAtTime(timestamp)

	// Create a slice of keys
	values := make([]resources.Resource, 0, len(m))
	for _, b := range m {
		r := resources.BytesToResource(b)
		if kind != "" && kind != r.Kind {
			continue
		}
		if namespace != "" && namespace != r.Namespace {
			continue
		}
		values = append(values, r)
	}

	return values
}
