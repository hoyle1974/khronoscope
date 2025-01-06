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
}

type Resource struct {
	Timestamp time.Time
	Kind      string
	Namespace string
	Name      string
	Object    any
	Extra     map[string]any
	Update    func() Resource
}

func NewResource(timestmap time.Time, kind string, namespace string, name string, obj any) Resource {
	return Resource{
		Timestamp: timestmap,
		Kind:      kind,
		Namespace: namespace,
		Name:      name,
		Object:    obj,
		Extra:     map[string]any{},
	}
}

func (r Resource) SetExtra(e map[string]any) Resource {
	r.Extra = e
	return r
}

func (r Resource) SetUpdate(u func() Resource) Resource {
	r.Update = u
	return r
}

func (r Resource) Key() string {
	return r.Kind + "/" + r.Name + "/" + r.Name
}

type Watcher struct {
	lock       sync.Mutex
	log        string
	lastChange time.Time
	timedMap   *TimedMap
	onChange   func()
	pool       pond.Pool
}

func (w *Watcher) Log(l string) {
	w.lock.Lock()
	w.log = l
	w.lock.Unlock()
}
func (w *Watcher) GetLog() string {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.log
}

func (w *Watcher) OnChange(onChange func()) {
	w.onChange = onChange
}

func (w Watcher) ChangedSince(t time.Time) bool {
	return w.lastChange.After(t)
}

func (w Watcher) GetStateAtTime(timestamp time.Time, kind string, namespace string) []Resource {
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

var ccc = 0

func (w *Watcher) startTick() {
	go func() {
		ccc++
		w.Log(fmt.Sprintf("%v", ccc))

		subpool := w.pool.NewSubpool(32)
		for _, r := range w.GetStateAtTime(time.Now(), "", "") {

			if r.Update != nil {
				subpool.Go(func() {
					r = r.Update()
					w.timedMap.Update(r.Timestamp, r.Key(), r)
				})
			}
		}
		subpool.StopAndWait()
		w.dirty()

		w.startTick()

	}()
}

func (w *Watcher) dirty() {
	w.lastChange = time.Now()
	if w.onChange != nil {
		w.onChange()
	}
}

func NewWatcher() *Watcher {
	w := &Watcher{
		lastChange: time.Now(),
		timedMap:   NewTimedMap(),
		pool:       pond.NewPool(64),
	}
	w.startTick()
	return w
}

func (w *Watcher) watchEvents(watcher <-chan watch.Event, watchMe WatchMe) {
	for {
		change := false

		if event, ok := <-watcher; ok {
			switch event.Type {
			case watch.Added:
				r := watchMe.Add(event.Object)
				w.timedMap.Add(r.Timestamp, r.Key(), r)
				change = true
			case watch.Modified:
				r := watchMe.Modified(event.Object)
				w.timedMap.Update(r.Timestamp, r.Key(), r)
				change = true
			case watch.Deleted:
				r := watchMe.Del(event.Object)
				w.timedMap.Remove(r.Timestamp, r.Key())
				change = true
			case watch.Error:
				fmt.Printf("Unknown error watching pods: %v\n", event.Object)
			}

			if change {
				w.dirty()
			}

		} else {
			fmt.Println("Channel closed")
			return
		}
	}
}
