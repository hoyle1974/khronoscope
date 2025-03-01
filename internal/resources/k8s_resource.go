package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/serializable"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type watcher struct {
	kind     string
	resource schema.GroupVersionResource
	renderer ResourceRenderer
	ticker   func()
}

func (g watcher) Tick() {
	if g.ticker != nil {
		g.ticker()
	}
}
func (g watcher) Kind() string               { return g.kind }
func (g watcher) Renderer() ResourceRenderer { return g.renderer }

func (g watcher) ToResource(obj runtime.Object) Resource {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return Resource{}
	}

	rawBytes, err := json.Marshal(unstructuredObj.Object)
	if err != nil {
		rawBytes = []byte("{}")
	}

	return Resource{
		Uid:       string(unstructuredObj.GetUID()),
		Timestamp: serializable.Time{Time: time.Now()},
		Kind:      unstructuredObj.GetKind(),
		Namespace: unstructuredObj.GetNamespace(),
		Name:      unstructuredObj.GetName(),
		RawJSON:   string(rawBytes),
		Extra:     nil,
	}
}

type defaultRenderer struct{}

func (r defaultRenderer) Render(resource Resource, details bool) []string {
	if details {
		s, _ := misc.PrettyPrintYAMLFromJSON(resource.RawJSON)
		return strings.Split(s, "\n")
	}
	return []string{resource.Name}
}

func watchForResource(ctx context.Context, watcher *K8sWatcher, k conn.KhronosConn, g watcher) error {
	watchChan, err := k.DynamicClient.Resource(g.resource).Watch(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to watch resource %s: %w", g.kind, err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), g)
	return nil
}
