package dao

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/hoyle1974/khronoscope/internal/misc"
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

type dataModelImpl struct {
	lock      sync.Mutex
	meta      temporal.Map
	resources temporal.Map
}

func New() KhronoStore {
	return &dataModelImpl{
		meta:      temporal.New(),
		resources: temporal.New(),
	}
}

func NewFromFile(filename string) KhronoStore {
	d := New().(*dataModelImpl)

	fi, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	// close on exit and check for its returned error
	defer func() {
		if err := fi.Close(); err != nil {
			panic(err)
		}
	}()

	reader := bufio.NewReader(fi)

	// Read the length of the first byte array
	var data1Len uint32
	if err := binary.Read(reader, binary.BigEndian, &data1Len); err != nil {
		panic(fmt.Errorf("failed to read data1 length: %w", err))
	}

	// Read the first byte array
	data1 := make([]byte, data1Len)
	if _, err := io.ReadFull(reader, data1); err != nil {
		panic(fmt.Errorf("failed to read data1: %w", err))
	}

	// Read the length of the second byte array
	var data2Len uint32
	if err := binary.Read(reader, binary.BigEndian, &data2Len); err != nil {
		panic(fmt.Errorf("failed to read data2 length: %w", err))
	}

	// Read the second byte array
	data2 := make([]byte, data2Len)
	if _, err := io.ReadFull(reader, data2); err != nil {
		panic(fmt.Errorf("failed to read data2: %w", err))
	}

	d.resources = temporal.FromBytes(data1)
	d.meta = temporal.FromBytes(data2)

	return d
}

func (d *dataModelImpl) Size() int {
	d.lock.Lock()
	defer d.lock.Unlock()

	resourceMap := d.resources.ToBytes()
	metaMap := d.meta.ToBytes()

	return len(resourceMap) + len(metaMap)
}

func (d *dataModelImpl) Save(filename string) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// open output file
	fo, err := os.Create(filename)
	if err != nil {
		panic(err)
	}

	// close on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	resourceMap := d.resources.ToBytes()
	metaMap := d.meta.ToBytes()

	writer := bufio.NewWriter(fo)

	// Write the length of the first byte array
	if err := binary.Write(writer, binary.BigEndian, uint32(len(resourceMap))); err != nil {
		panic(fmt.Errorf("failed to write data1 length: %w", err))
	}

	// Write the first byte array
	if _, err := writer.Write(resourceMap); err != nil {
		panic(fmt.Errorf("failed to write data1: %w", err))
	}

	// Write the length of the second byte array
	if err := binary.Write(writer, binary.BigEndian, uint32(len(metaMap))); err != nil {
		panic(fmt.Errorf("failed to write data2 length: %w", err))
	}

	// Write the second byte array
	if _, err := writer.Write(metaMap); err != nil {
		panic(fmt.Errorf("failed to write data2: %w", err))
	}

	if err := writer.Flush(); err != nil {
		panic(fmt.Errorf("failed to flush writer: %w", err))
	}
}

const META_LABEL_KEY = "Meta.Label"

func (d *dataModelImpl) SetLabel(time time.Time, label string) {
	d.meta.Add(time, META_LABEL_KEY, []byte(label))
}

func (d *dataModelImpl) GetLabel(time time.Time) string {
	currentLabel := ""
	if temp, ok := d.meta.GetStateAtTime(time)[META_LABEL_KEY]; ok {
		currentLabel = string(temp)
	}
	return currentLabel
}

func (d *dataModelImpl) GetNextLabelTime(time time.Time) time.Time {
	if t, err := d.meta.FindNextTimeKey(time, 1, META_LABEL_KEY); err == nil {
		return t
	}
	_, next := d.GetTimeRange()
	return next
}

func (d *dataModelImpl) GetPrevLabelTime(time time.Time) time.Time {
	if t, err := d.meta.FindNextTimeKey(time, -1, META_LABEL_KEY); err == nil {
		return t
	}
	prev, _ := d.GetTimeRange()
	return prev
}

func (d *dataModelImpl) GetTimeRange() (time.Time, time.Time) {
	return d.resources.GetTimeRange()
}

func (d *dataModelImpl) AddResource(resource resources.Resource) {
	log.Debug().Str("Resource", resource.Name).Str("Uid", resource.Uid).Msg("Add")

	d.lock.Lock()
	defer d.lock.Unlock()

	data, err := misc.EncodeToBytes(resource)
	if err != nil {
		panic(err)
	}

	d.resources.Add(resource.Timestamp.Time, resource.Key(), data)
}

func (d *dataModelImpl) UpdateResource(resource resources.Resource) {
	log.Debug().Str("Resource", resource.Name).Str("Uid", resource.Uid).Msg("Update")

	d.lock.Lock()
	defer d.lock.Unlock()

	data, err := misc.EncodeToBytes(resource)
	if err != nil {
		panic(err)
	}

	d.resources.Update(resource.Timestamp.Time, resource.Key(), data)

}

func (d *dataModelImpl) DeleteResource(resource resources.Resource) {
	log.Debug().Str("Resource", resource.Name).Str("Uid", resource.Uid).Msg("Delete")
	d.lock.Lock()
	defer d.lock.Unlock()

	d.resources.Remove(resource.Timestamp.Time, resource.Key())
}

func (d *dataModelImpl) GetResourceAt(timestamp time.Time, uid string) (resources.Resource, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	b := d.resources.GetItem(timestamp, uid)

	var r resources.Resource
	err := misc.DecodeFromBytes(b, &r)
	if err != nil {
		return r, err
	}

	return r, nil
}

func (d *dataModelImpl) GetResourcesAt(timestamp time.Time, kind string, namespace string) []resources.Resource {
	d.lock.Lock()
	defer d.lock.Unlock()
	m := d.resources.GetStateAtTime(timestamp)

	// Create a slice of keys
	values := make([]resources.Resource, 0, len(m))
	for _, v := range m {
		var r resources.Resource
		err := misc.DecodeFromBytes(v, &r)
		if err != nil {
			panic(fmt.Sprintf("Tried to decode %d bytes but got an error: %v", len(v), err))
		}
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
