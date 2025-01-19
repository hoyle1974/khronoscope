package main

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

const WATCHER_STEP = time.Second * 1

// Interface for watching resource events.
type ResourceEventWatcher interface {
	ToResource(obj runtime.Object) Resource // Converts a kubernetes object to a Resource
	Tick()                                  // Called at a regular interval and can be used to do any needed work to update Resources not handled by Add/Modified/Del like metrics
}

// Watches for a variety of k8s resources state changes and tracks their values over time
type K8sWatcher struct {
	lastChange  time.Time
	temporalMap *TemporalMap
	onChange    func()
}

// Create a new watcher
func NewK8sWatcher() *K8sWatcher {
	w := &K8sWatcher{
		lastChange:  time.Now(),
		temporalMap: NewTemporalMap(),
	}
	return w
}

// Set the onChange callback
func (w *K8sWatcher) OnChange(onChange func()) {
	w.onChange = onChange
}

// See if anything we watch has changed since a certain time
func (w *K8sWatcher) ChangedSince(t time.Time) bool {
	return w.lastChange.After(t)
}

// Returns a list of Resources that existed at a specific time, can be filtered by kind and namespace
func (w *K8sWatcher) GetStateAtTime(timestamp time.Time, kind string, namespace string) []Resource {
	m := w.temporalMap.GetStateAtTime(timestamp)

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

// Used internally to denote when the internal struct has been modified and notify anyone listening about that change
func (w *K8sWatcher) dirty() {
	w.lastChange = time.Now()
	if w.onChange != nil {
		w.onChange()
	}
}

// Add a resource to the temporal map
func (w *K8sWatcher) Add(r Resource) {
	w.temporalMap.Add(r.Timestamp, r.Key(), r)
	w.dirty()
}

// Update a resource in the temporal map
func (w *K8sWatcher) Update(r Resource) {
	w.temporalMap.Update(r.Timestamp, r.Key(), r)
	w.dirty()
}

// Delete a resource in the temporal map
func (w *K8sWatcher) Delete(r Resource) {
	w.temporalMap.Remove(r.Timestamp, r.Key())
	w.dirty()
}

func (w *K8sWatcher) registerEventWatcher(watcher <-chan watch.Event, resourceEventWatcher ResourceEventWatcher) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in goroutine: %v\n", r)
			panic(r) // Re-panic to crash
		}
	}()

	ticker := time.NewTicker(WATCHER_STEP)
	defer ticker.Stop()
	resourceEventWatcher.Tick()

	for {
		select {
		case event, ok := <-watcher:
			if !ok {
				fmt.Println("Channel closed")
				return
			}

			switch event.Type {
			case watch.Added:
				w.Add(resourceEventWatcher.ToResource(event.Object))
			case watch.Modified:
				w.Update(resourceEventWatcher.ToResource(event.Object))
			case watch.Deleted:
				w.Delete(resourceEventWatcher.ToResource(event.Object))
			case watch.Error:
				fmt.Printf("Unknown error watching: %v\n", event.Object)
			}
		case <-ticker.C:
			resourceEventWatcher.Tick()
		}
	}

}
