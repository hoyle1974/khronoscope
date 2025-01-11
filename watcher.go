package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/alitto/pond/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

type WatchMe interface {
	Valid(obj runtime.Object) bool
	Add(obj runtime.Object) Resource
	Modified(obj runtime.Object) Resource
	Del(obj runtime.Object) Resource
	Tick()
}

type K8sWatcher struct {
	lock       sync.Mutex
	log        string
	lastChange time.Time
	timedMap   *TimedMap
	onChange   func()
	pool       pond.Pool
}

func (w *K8sWatcher) Log(l string) {
	w.lock.Lock()
	w.log = l
	w.lock.Unlock()
}
func (w *K8sWatcher) GetLog() string {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.log
}

func (w *K8sWatcher) OnChange(onChange func()) {
	w.onChange = onChange
}

func (w K8sWatcher) ChangedSince(t time.Time) bool {
	return w.lastChange.After(t)
}

func (w K8sWatcher) GetStateAtTime(timestamp time.Time, kind string, namespace string) []Resource {
	m := w.timedMap.GetStateAtTime(timestamp)

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

func (w *K8sWatcher) dirty() {
	w.lastChange = time.Now()
	if w.onChange != nil {
		w.onChange()
	}
}

func NewK8sWatcher() *K8sWatcher {
	w := &K8sWatcher{
		lastChange: time.Now(),
		timedMap:   NewTimedMap(),
		pool:       pond.NewPool(64),
	}
	return w
}

func (w *K8sWatcher) Add(r Resource) {
	w.timedMap.Add(r.Timestamp, r.Key(), r)
	w.dirty()
}

func (w *K8sWatcher) Update(r Resource) {
	w.timedMap.Update(r.Timestamp, r.Key(), r)
	w.dirty()
}

func (w *K8sWatcher) Delete(r Resource) {
	w.timedMap.Remove(r.Timestamp, r.Key())
	w.dirty()
}

func (w *K8sWatcher) watchEvents(watcher <-chan watch.Event, watchMe WatchMe) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in goroutine: %v\n", r)
			panic(r) // Re-panic to crash
		}
	}()

	ticker := time.NewTicker(time.Second / 2)
	defer ticker.Stop()
	watchMe.Tick()

	for {
		select {
		case event, ok := <-watcher:
			if !ok {
				fmt.Println("Channel closed")
				return
			}

			switch event.Type {
			case watch.Added:
				// w.pool.Go(func() {
				w.Add(watchMe.Add(event.Object))
				// })
			case watch.Modified:
				// w.pool.Go(func() {
				w.Update(watchMe.Modified(event.Object))
				// })
			case watch.Deleted:
				// w.pool.Go(func() {
				w.Delete(watchMe.Del(event.Object))
				// })
			case watch.Error:
				fmt.Printf("Unknown error watching: %v\n", event.Object)
			}
		case <-ticker.C:
			watchMe.Tick()
		}
	}

}
