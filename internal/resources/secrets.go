package resources

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type SecretExtra struct {
	Name              string
	Namespace         string
	UID               string
	CreationTimestamp time.Time
	Labels            []string
	Annotations       []string
	DataKeys          []string
}

func (p SecretExtra) Copy() Copyable {
	return SecretExtra{
		Name:              p.Name,
		Namespace:         p.Namespace,
		UID:               p.UID,
		CreationTimestamp: p.CreationTimestamp,
		Labels:            misc.DeepCopyArray(p.Labels),
		Annotations:       misc.DeepCopyArray(p.Annotations),
		DataKeys:          misc.DeepCopyArray(p.DataKeys),
	}
}

func newSecretExtra(secret *corev1.Secret) SecretExtra {
	labels := misc.RenderMapOfStrings(secret.Labels)
	annotations := misc.RenderMapOfStrings(secret.Annotations)

	dataKeys := make([]string, 0, len(secret.Data))
	for k := range secret.Data {
		dataKeys = append(dataKeys, k)
	}

	sort.Strings(dataKeys)

	return SecretExtra{
		Name:              secret.Name,
		Namespace:         secret.Namespace,
		UID:               string(secret.UID),
		CreationTimestamp: secret.CreationTimestamp.Time,
		Labels:            labels,
		Annotations:       annotations,
		DataKeys:          dataKeys,
	}
}

func renderSecretExtra(extra SecretExtra) []string {
	output := []string{
		fmt.Sprintf("Name: %s", extra.Name),
		fmt.Sprintf("Namespace: %s", extra.Namespace),
		fmt.Sprintf("UID: %s", extra.UID),
		fmt.Sprintf("CreationTimestamp: %s", extra.CreationTimestamp.Format(time.RFC3339)),
	}
	output = append(output, extra.Labels...)
	output = append(output, extra.Annotations...)
	output = append(output, fmt.Sprintf("Data keys: %v", extra.DataKeys))
	return output
}

type SecretRenderer struct {
}

func (r SecretRenderer) Render(resource Resource, details bool) []string {
	extra := resource.Extra.(SecretExtra)

	if details {
		return renderSecretExtra(extra)
	}

	return []string{resource.Key()}
}

type SecretWatcher struct {
}

func (n SecretWatcher) Tick() {
}

func (n SecretWatcher) Kind() string {
	return "Secret"
}

func (n SecretWatcher) Renderer() ResourceRenderer {
	return SecretRenderer{}
}

func (n SecretWatcher) convert(obj runtime.Object) *corev1.Secret {
	ret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}
	return ret
}

func (n SecretWatcher) ToResource(obj runtime.Object) Resource {
	secret := n.convert(obj)
	extra := newSecretExtra(secret)
	return NewK8sResource(n.Kind(), secret, extra)
}

func watchForSecret(watcher *K8sWatcher, k conn.KhronosConn, ns string) error {
	watchChan, err := k.Client.CoreV1().Secrets(ns).Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), SecretWatcher{})

	return nil
}
