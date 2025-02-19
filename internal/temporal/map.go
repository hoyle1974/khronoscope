package temporal

import (
	"bytes"
	"encoding/gob"
	"errors"
	"sync"
	"time"

	"github.com/hoyle1974/khronoscope/internal/serializable"
)

// TemporalMap acts like a map with the added ability to rewind and see a snapshot of the state of
// the map given a time.Time.  Internally it is implemented as a map[string]*TemporalValue where
// TemporalValue a sorted array of TemporalValuePairs. Once a key/value is inserted into a TemporalMap
// it can never truly be deleted. It's just inserted as a new temporalValuePair
// with an updated timestamp and a value of nil.  I'm added support internally for Keyframes and binary diffs.
//
// TemporalMap is threadsafe.
//


type Map interface {
	ToBytes() []byte
	GetTimeRange() (time.Time, time.Time)
	Add(timestamp time.Time, key string, value []byte)
	GetItem(timestamp time.Time, key string) []byte
	Update(timestamp time.Time, key string, value []byte)
	Remove(timestamp time.Time, key string)
	GetStateAtTime(timestamp time.Time) map[string][]byte
	FindNextTimeKey(timestamp time.Time, dir int, key string) (time.Time, error)
}

// Map represents a map-like data structure with time-ordered items.
type mapImpl struct {
	lock    sync.RWMutex
	Items   map[string]*TimeValueStore
	MinTime serializable.Time
	MaxTime serializable.Time
}

// New creates a new empty TimedMap.
func New() Map {
	return &mapImpl{
		Items: map[string]*TimeValueStore{},
	}
}

func FromBytes(b []byte) Map {
	dec := gob.NewDecoder(bytes.NewReader(b))

	tm := New().(*mapImpl)
	err := dec.Decode(&tm)
	if err != nil {
		panic(err)
	}

	return tm
}

func (tm *mapImpl) ToBytes() []byte {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	var network bytes.Buffer

	enc := gob.NewEncoder(&network)
	if err := enc.Encode(tm); err != nil {
		panic(err)
	}

	return network.Bytes()
}

func (tm *mapImpl) GetTimeRange() (time.Time, time.Time) {
	return tm.MinTime.Time, tm.MaxTime.Time
}

func (tm *mapImpl) updateTimeRange(timestamp time.Time) {
	if len(tm.Items) == 0 || timestamp.Before(tm.MinTime.Time) {
		tm.MinTime = serializable.Time{Time: timestamp}
	}
	if len(tm.Items) == 0 || timestamp.After(tm.MaxTime.Time) {
		tm.MaxTime = serializable.Time{Time: timestamp}
	}

}

// Add adds an item to the TimedMap with the given timestamp, key, and value.
func (tm *mapImpl) Add(timestamp time.Time, key string, value []byte) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.updateTimeRange(timestamp)

	v, ok := tm.Items[key]
	if !ok {
		v = NewTimeValueStore()
	}
	v.AddValue(timestamp, value)
	tm.Items[key] = v
}

func (tm *mapImpl) GetItem(timestamp time.Time, key string) []byte {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if item, ok := tm.Items[key]; ok {
		return item.QueryValue(timestamp)
	}

	return nil
}

// Update the value of an item with the given timestamp and key.
func (tm *mapImpl) Update(timestamp time.Time, key string, value []byte) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.updateTimeRange(timestamp)

	v, ok := tm.Items[key]
	if !ok {
		v = NewTimeValueStore()
	}
	//if v.QueryValue(timestamp) != nil {
	v.AddValue(timestamp, value)
	tm.Items[key] = v
	//}
}

// Remove removes the item with the given timestamp and key.
func (tm *mapImpl) Remove(timestamp time.Time, key string) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.updateTimeRange(timestamp)

	v, ok := tm.Items[key]
	if !ok {
		v = NewTimeValueStore()
	}
	v.AddValue(timestamp, nil)
	tm.Items[key] = v
}

// GetStateAtTime returns a map of key-value pairs at the given timestamp.
func (tm *mapImpl) GetStateAtTime(timestamp time.Time) map[string][]byte {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	state := make(map[string][]byte)
	for key, item := range tm.Items {
		value := item.QueryValue(timestamp)
		if len(value) > 0 {
			state[key] = value
		}
	}
	return state
}

func (tm *mapImpl) FindNextTimeKey(timestamp time.Time, dir int, key string) (time.Time, error) {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	if tvs, ok := tm.Items[key]; ok {
		return tvs.FindNextTimeKey(timestamp, dir)
	}

	return timestamp, errors.New("no key found")
}
