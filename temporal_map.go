package main

import (
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
	history []temporalValuePair
}

type temporalValuePair struct {
	Timestamp time.Time
	Value     any
}

// Set the value at the specific time, maintaining chronological order.
func (i *temporalValue) Set(timestamp time.Time, value any) {
	// Find the insertion point to maintain chronological order.
	index := sort.Search(len(i.history), func(j int) bool {
		return i.history[j].Timestamp.After(timestamp)
	})

	// Insert the new value at the correct position.
	i.history = append(i.history[:index], append([]temporalValuePair{{Timestamp: timestamp, Value: value}}, i.history[index:]...)...)
}

// Get the value at a specific time.
func (i *temporalValue) Get(timestamp time.Time) any {
	// Find the most recent value before or at the given timestamp
	for j := len(i.history) - 1; j >= 0; j-- {
		if i.history[j].Timestamp.Before(timestamp) || i.history[j].Timestamp.Equal(timestamp) {
			return i.history[j].Value
		}
	}
	return nil // No value found before or at the given timestamp
}

// TemporalMap represents a map-like data structure with time-ordered items.
type TemporalMap struct {
	lock    sync.RWMutex
	items   map[string]*temporalValue
	minTime time.Time
	maxTime time.Time
}

// NewTemporalMap creates a new empty TimedMap.
func NewTemporalMap() *TemporalMap {
	return &TemporalMap{
		items: map[string]*temporalValue{},
	}
}

func (tm *TemporalMap) GetTimeRange() (time.Time, time.Time) {
	return tm.minTime, tm.maxTime
}

func (tm *TemporalMap) ClampTime(t time.Time) time.Time {
	if t.Before(tm.minTime) {
		t = tm.minTime
	}
	if t.After(tm.maxTime) {
		t = tm.maxTime
	}
	return t
}

func (tm *TemporalMap) updateTimeRange(timestamp time.Time) {
	if len(tm.items) == 0 || timestamp.Before(tm.minTime) {
		tm.minTime = timestamp
	}
	if len(tm.items) == 0 || timestamp.After(tm.maxTime) {
		tm.maxTime = timestamp
	}

}

// Add adds an item to the TimedMap with the given timestamp, key, and value.
func (tm *TemporalMap) Add(timestamp time.Time, key string, value interface{}) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.updateTimeRange(timestamp)

	v, ok := tm.items[key]
	if !ok {
		v = &temporalValue{}
	}
	v.Set(timestamp, value)
	tm.items[key] = v
}

// Update updates the value of an item with the given timestamp and key.
func (tm *TemporalMap) Update(timestamp time.Time, key string, value interface{}) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.updateTimeRange(timestamp)

	v, ok := tm.items[key]
	if !ok {
		v = &temporalValue{}
	}
	if v.Get(timestamp) != nil {
		v.Set(timestamp, value)
		tm.items[key] = v
	}
}

// Remove removes the item with the given timestamp and key.
func (tm *TemporalMap) Remove(timestamp time.Time, key string) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.updateTimeRange(timestamp)

	v, ok := tm.items[key]
	if !ok {
		v = &temporalValue{}
	}
	v.Set(timestamp, nil)
	tm.items[key] = v
}

// GetStateAtTime returns a map of key-value pairs at the given timestamp.
func (tm *TemporalMap) GetStateAtTime(timestamp time.Time) map[string]interface{} {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	state := make(map[string]interface{})
	for key, item := range tm.items {
		value := item.Get(timestamp)
		if value != nil {
			state[key] = value
		}
	}
	return state
}
