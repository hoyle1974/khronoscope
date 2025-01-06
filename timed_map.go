package main

import (
	"sort"
	"sync"
	"time"
)

type TemporalValue struct {
	history []ValuePair
}

type ValuePair struct {
	Timestamp time.Time
	Value     any
}

// // Set the value at the specific time
//
//	func (i *TemporalValue) Set(timestamp time.Time, value any) {
//		i.history = append(i.history, ValuePair{
//			Timestamp: timestamp,
//			Value:     value,
//		})
//	}
//
// Set the value at the specific time, maintaining chronological order.
func (i *TemporalValue) Set(timestamp time.Time, value any) {
	// Find the insertion point to maintain chronological order.
	index := sort.Search(len(i.history), func(j int) bool {
		return i.history[j].Timestamp.After(timestamp)
	})

	// Insert the new value at the correct position.
	i.history = append(i.history[:index], append([]ValuePair{{Timestamp: timestamp, Value: value}}, i.history[index:]...)...)
}

// Get the value at a specific time.
func (i *TemporalValue) Get(timestamp time.Time) any {
	// Find the most recent value before or at the given timestamp
	for j := len(i.history) - 1; j >= 0; j-- {
		if i.history[j].Timestamp.Before(timestamp) || i.history[j].Timestamp.Equal(timestamp) {
			return i.history[j].Value
		}
	}
	return nil // No value found before or at the given timestamp
}

// TimedMap represents a map-like data structure with time-ordered items.
type TimedMap struct {
	lock  sync.RWMutex
	items map[string]*TemporalValue
}

// NewTimedMap creates a new empty TimedMap.
func NewTimedMap() *TimedMap {
	return &TimedMap{
		items: map[string]*TemporalValue{},
	}
}

// Add adds an item to the TimedMap with the given timestamp, key, and value.
func (tm *TimedMap) Add(timestamp time.Time, key string, value interface{}) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	v, ok := tm.items[key]
	if !ok {
		v = &TemporalValue{}
	}
	v.Set(timestamp, value)
	tm.items[key] = v
}

// Update updates the value of an item with the given timestamp and key.
func (tm *TimedMap) Update(timestamp time.Time, key string, value interface{}) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	v, ok := tm.items[key]
	if !ok {
		v = &TemporalValue{}
	}
	if v.Get(timestamp) != nil {
		v.Set(timestamp, value)
		tm.items[key] = v
	}
}

// Remove removes the item with the given timestamp and key.
func (tm *TimedMap) Remove(timestamp time.Time, key string) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	v, ok := tm.items[key]
	if !ok {
		v = &TemporalValue{}
	}
	v.Set(timestamp, nil)
	tm.items[key] = v
}

// GetStateAtTime returns a map of key-value pairs at the given timestamp.
func (tm *TimedMap) GetStateAtTime(timestamp time.Time) map[string]interface{} {
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
