package resources

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/hoyle1974/khronoscope/internal/config"
	"github.com/hoyle1974/khronoscope/internal/conn"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

const WATCHER_STEP = time.Second * 1

type DAO interface {
	AddResource(Resource)
	UpdateResource(Resource)
	DeleteResource(Resource)
	GetResourcesAt(timestamp time.Time, kind string, namespace string) []Resource
	GetResourceAt(timestamp time.Time, uid string) (Resource, error)
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

func (w *K8sWatcher) StartWatching(ctx context.Context, client conn.KhronosConn, dao DAO, lc *LogCollector, ns string) error {
	// Get API group resources
	apiGroupResources, err := client.DiscoveryClient.ServerPreferredResources()
	if err != nil {
		log.Warn().Err(err).Msg("Warning: Some API groups may not be accessible")
	}

	filter := config.Get().Filter

	// Extract GroupVersionResource information
	for _, list := range apiGroupResources {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			log.Warn().Err(err).Any("GroupVersion", list.GroupVersion).Msg("Skipping invalid GroupVersion:")
			continue
		}

		for _, resource := range list.APIResources {
			if filter.Standard && (resource.Kind == "" || (!resource.Namespaced && resource.Kind != "Namespace")) {
				continue
			}

			gvr := schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: resource.Name,
			}

			var ticker func()
			var renderer ResourceRenderer

			if resource.Kind == "Node" {
				ticker = func() {
					NodeTicker(dao, client.MetricsClient)
				}
				renderer = NodeRenderer{dao: dao}
			} else if resource.Kind == "Pod" {
				ticker = func() {
					PodTicker(dao, client.MetricsClient)
				}
				renderer = PodRenderer{dao: dao}
			} else {
				renderer = GenericRenderer{}
			}

			if err := watchForResource(ctx, w, client, GenericWatcher{kind: resource.Kind, resource: gvr, renderer: renderer, ticker: ticker}); err != nil {
				fmt.Printf("	error:%v", err)
			}
		}
	}

	return nil
}

// Create a new watcher
var (
	_watcher    *K8sWatcher
	onceWatcher sync.Once
)

func GetK8sWatcher(data DAO) *K8sWatcher {
	onceWatcher.Do(func() {
		_watcher = &K8sWatcher{
			lastChange: time.Now(),
			data:       data,
		}
	})
	return _watcher
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
	log.Info().Any("Watcher", reflect.TypeOf(resourceEventWatcher)).Msg("registerEventWatcher")

	RegisterResourceRenderer(resourceEventWatcher.Kind(), resourceEventWatcher.Renderer())
	if w == nil {
		return
	}

	// defer func() {
	// 	if r := recover(); r != nil {
	// 		fmt.Printf("Recovered in goroutine: %v\n", r)
	// 		panic(r) // Re-panic to crash
	// 	}
	// }()

	ticker := time.NewTicker(WATCHER_STEP)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-watcher:
			if !ok {
				// fmt.Println("Channel closed")
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
				// fmt.Printf("Unknown error watching: %v\n", event.Object)
			}
		case <-ticker.C:
			resourceEventWatcher.Tick()
		}
	}

}
