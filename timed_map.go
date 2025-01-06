package main

import (
	"sync"
	"time"
)

// The TemporalValue effectively stores a single value in it.  But tracks the values changes
// over time.  The value may be nil.  You may request the value at a specific time and it
// will return what that value was.  For example if you ask for the value at a timestamp
// before you ever insert it, then it will return nil.  If you set the value at noon, and then
// Ask what the value is at 12:05 it tells you the original value.  If you set a new value at 12:10
// And ask for the value at 12:05 it gives you the original value.  If you ask what the value was at
// 12:11 it give you the updated value.
// type TemporalValue struct {
// }

// // Set the value at the specific time
// func (i *TemporalValue) Set(timestamp time.Time, value any) {
// }

// // Get the value at a specific time.
// func (i *TemporalValue) Get(timestamp time.Time) any {
// 	return nil
// }

type TemporalValue struct {
	history []ValuePair
}

type ValuePair struct {
	Timestamp time.Time
	Value     any
}

// Set the value at the specific time
func (i *TemporalValue) Set(timestamp time.Time, value any) {
	i.history = append(i.history, ValuePair{
		Timestamp: timestamp,
		Value:     value,
	})
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
	v.Set(timestamp, value)
	tm.items[key] = v
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
