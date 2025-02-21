package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/serializable"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type GenericExtra struct {
	RawJSON string
	Age     serializable.Time
}

func (g GenericExtra) Copy() Copyable {
	return GenericExtra{
		RawJSON: g.RawJSON,
		Age:     g.Age,
	}
}

type GenericWatcher struct {
	kind     string
	resource schema.GroupVersionResource
}

func NewGenericWatcher(kind string, group, version, resource string) GenericWatcher {
	return GenericWatcher{
		kind: kind,
		resource: schema.GroupVersionResource{
			Group:    group,
			Version:  version,
			Resource: resource,
		},
	}
}

func (g GenericWatcher) Tick()                      {}
func (g GenericWatcher) Kind() string               { return g.kind }
func (g GenericWatcher) Renderer() ResourceRenderer { return GenericRenderer{} }

func (g GenericWatcher) ToResource(obj runtime.Object) Resource {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return Resource{}
	}

	rawBytes, err := json.Marshal(unstructuredObj.Object)
	if err != nil {
		rawBytes = []byte("{}")
	}

	extra := GenericExtra{
		RawJSON: string(rawBytes),
		Age:     serializable.NewTime(unstructuredObj.GetCreationTimestamp().Time),
	}

	uid := string(unstructuredObj.GetUID())
	name := unstructuredObj.GetName()
	namespace := unstructuredObj.GetNamespace()
	return NewK8sResource2(g.Kind(), uid, namespace, name, extra)
}

func PrettyPrintJSON(jsonStr string) (string, error) {
	var jsonData map[string]interface{}

	// Unmarshal the JSON string into a map
	err := json.Unmarshal([]byte(jsonStr), &jsonData)
	if err != nil {
		return "", err
	}

	// Marshal it back with indentation
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return "", err
	}

	return string(prettyJSON), nil
}

func PrettyPrintYAMLFromJSON(jsonStr string) (string, error) {
	var jsonData interface{}

	// Decode JSON string
	err := json.Unmarshal([]byte(jsonStr), &jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Convert JSON to YAML
	yamlData, err := yaml.Marshal(jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %v", err)
	}

	return string(yamlData), nil
}

type GenericRenderer struct{}

func (r GenericRenderer) Render(resource Resource, details bool) []string {
	extra, ok := resource.Extra.(GenericExtra)
	if !ok {
		return []string{"[Invalid Resource]"}
	}
	if details {
		s, _ := PrettyPrintYAMLFromJSON(extra.RawJSON)
		return strings.Split(s, "\n")
	}
	return []string{resource.Name}
	// []string{extra.RawJSON} // Mimics `kubectl` raw output
}

func watchForResource(ctx context.Context, watcher *K8sWatcher, k conn.KhronosConn, g GenericWatcher) error {
	watchChan, err := k.DynamicClient.Resource(g.resource).Watch(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to watch resource %s: %w", g.kind, err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), g)
	return nil
}
