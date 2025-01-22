package main

import (
	"bytes"
	"encoding/gob"
	"sort"
	"sync"
	"time"
)

// TemporalMap acts like a map with the added ability to rewind and see a snapshot of the state of the map given a time.Time
// Internally it is implemented as a map[string]*TemporalValue where TemporalValue a sorted array of TemporalValuePairs.
// Once a key/value is inserted into a TemporalMap it can never truly be deleted. It's just inserted as a new temporalValuePair
// with an updated timestamp and a value of nil.
//
// TemporalMap is threadsafe.

type temporalValue struct {
	History []temporalValuePair
}

type temporalValuePair struct {
	Timestamp time.Time
	Value     any
}

// Set the value at the specific time, maintaining chronological order.
func (i *temporalValue) Set(timestamp time.Time, value any) {
	// Find the insertion point to maintain chronological order.
	index := sort.Search(len(i.History), func(j int) bool {
		return i.History[j].Timestamp.After(timestamp)
	})

	// Insert the new value at the correct position.
	i.History = append(i.History[:index], append([]temporalValuePair{{Timestamp: timestamp, Value: value}}, i.History[index:]...)...)
}

// Get the value at a specific time.
func (i *temporalValue) Get(timestamp time.Time) any {
	// Find the most recent value before or at the given timestamp
	for j := len(i.History) - 1; j >= 0; j-- {
		if i.History[j].Timestamp.Before(timestamp) || i.History[j].Timestamp.Equal(timestamp) {
			return i.History[j].Value
		}
	}
	return nil // No value found before or at the given timestamp
}

// TemporalMap represents a map-like data structure with time-ordered items.
type TemporalMap struct {
	lock    sync.RWMutex
	Items   map[string]*temporalValue
	MinTime time.Time
	MaxTime time.Time
}

// NewTemporalMap creates a new empty TimedMap.
func NewTemporalMap() *TemporalMap {
	return &TemporalMap{
		Items: map[string]*temporalValue{},
	}
}

func NewTemporalMapFromBytes(b []byte) *TemporalMap {
	dec := gob.NewDecoder(bytes.NewReader(b))

	tm := NewTemporalMap()
	dec.Decode(&tm)

	return tm
}

func (tm *TemporalMap) ToBytes() []byte {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	var network bytes.Buffer

	enc := gob.NewEncoder(&network)
	if err := enc.Encode(tm); err != nil {
		panic(err)
	}

	return network.Bytes()
}

func (tm *TemporalMap) GetTimeRange() (time.Time, time.Time) {
	return tm.MinTime, tm.MaxTime
}

func (tm *TemporalMap) ClampTime(t time.Time) time.Time {
	if t.Before(tm.MinTime) {
		t = tm.MinTime
	}
	if t.After(tm.MaxTime) {
		t = tm.MaxTime
	}
	return t
}

func (tm *TemporalMap) updateTimeRange(timestamp time.Time) {
	if len(tm.Items) == 0 || timestamp.Before(tm.MinTime) {
		tm.MinTime = timestamp
	}
	if len(tm.Items) == 0 || timestamp.After(tm.MaxTime) {
		tm.MaxTime = timestamp
	}

}

// Add adds an item to the TimedMap with the given timestamp, key, and value.
func (tm *TemporalMap) Add(timestamp time.Time, key string, value interface{}) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.updateTimeRange(timestamp)

	v, ok := tm.Items[key]
	if !ok {
		v = &temporalValue{}
	}
	v.Set(timestamp, value)
	tm.Items[key] = v
}

// Update updates the value of an item with the given timestamp and key.
func (tm *TemporalMap) Update(timestamp time.Time, key string, value interface{}) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.updateTimeRange(timestamp)

	v, ok := tm.Items[key]
	if !ok {
		v = &temporalValue{}
	}
	if v.Get(timestamp) != nil {
		v.Set(timestamp, value)
		tm.Items[key] = v
	}
}

// Remove removes the item with the given timestamp and key.
func (tm *TemporalMap) Remove(timestamp time.Time, key string) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.updateTimeRange(timestamp)

	v, ok := tm.Items[key]
	if !ok {
		v = &temporalValue{}
	}
	v.Set(timestamp, nil)
	tm.Items[key] = v
}

// GetStateAtTime returns a map of key-value pairs at the given timestamp.
func (tm *TemporalMap) GetStateAtTime(timestamp time.Time) map[string]interface{} {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	state := make(map[string]interface{})
	for key, item := range tm.Items {
		value := item.Get(timestamp)
		if value != nil {
			state[key] = value
		}
	}
	return state
}
