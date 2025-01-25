package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/hoyle1974/khronoscope/internal/temporal"
)

type DataModel interface {
	GetResourcesAt(timestamp time.Time, kind string, namespace string) []Resource
	GetTimeRange() (time.Time, time.Time)
	AddResource(resource Resource)
	UpdateResource(resource Resource)
	DeleteResource(resource Resource)
	SetLabel(time.Time, string)
	GetLabel(time time.Time) string
	Save(string)
}

type dataModelImpl struct {
	lock      sync.Mutex
	meta      temporal.Map
	resources temporal.Map
}

func NewDataModel() DataModel {
	return &dataModelImpl{
		meta:      temporal.New(),
		resources: temporal.New(),
	}
}

func NewDataModelFromFile(filename string) DataModel {
	d := NewDataModel().(*dataModelImpl)

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

func (d *dataModelImpl) SetLabel(time time.Time, label string) {
	d.meta.Add(time, "Meta.Label", label)
}

func (d *dataModelImpl) GetLabel(time time.Time) string {
	currentLabel := ""
	if temp, ok := d.meta.GetStateAtTime(time)["Meta.Label"]; ok {
		currentLabel = temp.(string)
	}
	return currentLabel
}

func (d *dataModelImpl) GetTimeRange() (time.Time, time.Time) {
	return d.resources.GetTimeRange()
}

func (d *dataModelImpl) AddResource(resource Resource) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.resources.Add(resource.Timestamp.Time, resource.Key(), resource)
}

func (d *dataModelImpl) UpdateResource(resource Resource) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.resources.Update(resource.Timestamp.Time, resource.Key(), resource)

}

func (d *dataModelImpl) DeleteResource(resource Resource) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.resources.Remove(resource.Timestamp.Time, resource.Key())
}

func (d *dataModelImpl) GetResourcesAt(timestamp time.Time, kind string, namespace string) []Resource {
	d.lock.Lock()
	defer d.lock.Unlock()
	m := d.resources.GetStateAtTime(timestamp)

	// Create a slice of keys
	values := make([]Resource, 0, len(m))
	for _, v := range m {
		r := v.(Resource)
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
