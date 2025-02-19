package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type PersistentVolumeExtra struct {
	Name                          string
	Namespace                     string
	UID                           string
	CreationTimestamp             time.Time
	Labels                        []string
	Annotations                   []string
	Capacity                      string
	AccessModes                   []string
	PersistentVolumeReclaimPolicy string
}

func (p PersistentVolumeExtra) Copy() Copyable {
	return PersistentVolumeExtra{
		Name:                          p.Name,
		Namespace:                     p.Namespace,
		UID:                           p.UID,
		CreationTimestamp:             p.CreationTimestamp,
		Labels:                        misc.DeepCopyArray(p.Labels),
		Annotations:                   misc.DeepCopyArray(p.Annotations),
		Capacity:                      p.Capacity,
		AccessModes:                   misc.DeepCopyArray(p.AccessModes),
		PersistentVolumeReclaimPolicy: p.PersistentVolumeReclaimPolicy,
	}
}

func newPersistentVolumeExtra(pv *corev1.PersistentVolume) PersistentVolumeExtra {
	labels := misc.RenderMapOfStrings(pv.Labels)
	annotations := misc.RenderMapOfStrings(pv.Annotations)

	// Get access modes
	accessModes := make([]string, 0, len(pv.Spec.AccessModes))
	for _, mode := range pv.Spec.AccessModes {
		accessModes = append(accessModes, string(mode))
	}

	capacity := fmt.Sprintf("%v", pv.Spec.Capacity[corev1.ResourceStorage])

	return PersistentVolumeExtra{
		Name:                          pv.Name,
		Namespace:                     pv.Namespace,
		UID:                           string(pv.UID),
		CreationTimestamp:             pv.CreationTimestamp.Time,
		Labels:                        labels,
		Annotations:                   annotations,
		Capacity:                      capacity,
		AccessModes:                   accessModes,
		PersistentVolumeReclaimPolicy: string(pv.Spec.PersistentVolumeReclaimPolicy),
	}
}

func renderPersistentVolumeExtra(extra PersistentVolumeExtra) []string {
	output := []string{
		fmt.Sprintf("Name: %s", extra.Name),
		fmt.Sprintf("Namespace: %s", extra.Namespace),
		fmt.Sprintf("UID: %s", extra.UID),
		fmt.Sprintf("CreationTimestamp: %s", extra.CreationTimestamp.Format(time.RFC3339)),
		fmt.Sprintf("Capacity: %s", extra.Capacity),
		fmt.Sprintf("Access Modes: %v", extra.AccessModes),
		fmt.Sprintf("Reclaim Policy: %s", extra.PersistentVolumeReclaimPolicy),
	}
	output = append(output, extra.Labels...)
	output = append(output, extra.Annotations...)
	return output
}

type PersistentVolumeRenderer struct {
}

func (r PersistentVolumeRenderer) Render(resource Resource, details bool) []string {
	extra := resource.Extra.(PersistentVolumeExtra)

	if details {
		return renderPersistentVolumeExtra(extra)
	}

	return []string{resource.Key()}
}

type PersistentVolumeWatcher struct {
}

func (n PersistentVolumeWatcher) Tick() {
}

func (n PersistentVolumeWatcher) Kind() string {
	return "PersistentVolume"
}

func (n PersistentVolumeWatcher) Renderer() ResourceRenderer {
	return PersistentVolumeRenderer{}
}

func (n PersistentVolumeWatcher) convert(obj runtime.Object) *corev1.PersistentVolume {
	ret, ok := obj.(*corev1.PersistentVolume)
	if !ok {
		return nil
	}
	return ret
}

func (n PersistentVolumeWatcher) ToResource(obj runtime.Object) Resource {
	pv := n.convert(obj)
	extra := newPersistentVolumeExtra(pv)
	return NewK8sResource(n.Kind(), pv, extra)
}

// watchForPersistentVolume watches for PersistentVolume events
func watchForPersistentVolume(watcher *K8sWatcher, k conn.KhronosConn) error {
	// Watch for PersistentVolume events
	watchChan, err := k.Client.CoreV1().PersistentVolumes().Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), PersistentVolumeWatcher{})

	return nil
}
