package resources

import (
	"fmt"
	"time"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

const WATCHER_STEP = time.Second * 1

type DAO interface {
	AddResource(Resource)
	UpdateResource(Resource)
	DeleteResource(Resource)
	GetResourcesAt(timestamp time.Time, kind string, namespace string) []Resource
}

// Interface for watching resource events.
type ResourceEventWatcher interface {
	ToResource(obj runtime.Object) Resource // Converts a kubernetes object to a Resource
	Tick()                                  // Called at a regular interval and can be used to do any needed work to update Resources not handled by Add/Modified/Del like metrics
	Renderer() ResourceRenderer
	Kind() string
}

// Watches for a variety of k8s resources state changes and tracks their values over time
type K8sWatcher struct {
	lastChange time.Time
	data       DAO
	onChange   func()
}

func (w *K8sWatcher) Watch(client conn.KhronosConn, dao DAO, lc *LogCollector) error {
	if err := watchForDeployments(w, client); err != nil {
		return err
	}
	if err := watchForDaemonSet(w, client); err != nil {
		return err
	}
	if err := watchForReplicaSet(w, client); err != nil {
		return err
	}
	if err := watchForService(w, client); err != nil {
		return err
	}
	if err := watchForNamespaces(w, client); err != nil {
		return err
	}
	podWatcher, err := watchForPods(w, client, dao, lc)
	if err != nil {
		return err
	}
	if _, err = watchForNodes(w, client, dao, podWatcher); err != nil {
		return err
	}
	return nil
}

// Create a new watcher
func NewK8sWatcher(data DAO) *K8sWatcher {
	w := &K8sWatcher{
		lastChange: time.Now(),
		data:       data,
	}
	return w
}

// Set the onChange callback
func (w *K8sWatcher) OnChange(onChange func()) {
	if w == nil {
		return
	}
	w.onChange = onChange
}

// See if anything we watch has changed since a certain time
func (w *K8sWatcher) ChangedSince(t time.Time) bool {
	return w.lastChange.After(t)
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
	w.data.AddResource(r)
	w.dirty()
}

// Update a resource in the temporal map
func (w *K8sWatcher) Update(r Resource) {
	w.data.UpdateResource(r)
	w.dirty()
}

// Delete a resource in the temporal map
func (w *K8sWatcher) Delete(r Resource) {
	w.data.DeleteResource(r)
	w.dirty()
}

func (w *K8sWatcher) registerEventWatcher(watcher <-chan watch.Event, resourceEventWatcher ResourceEventWatcher) {
	RegisterResourceRenderer(resourceEventWatcher.Kind(), resourceEventWatcher.Renderer())
	if w == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in goroutine: %v\n", r)
			panic(r) // Re-panic to crash
		}
	}()

	ticker := time.NewTicker(WATCHER_STEP)
	defer ticker.Stop()

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
