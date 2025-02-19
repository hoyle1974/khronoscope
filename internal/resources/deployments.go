package resources

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DeploymentExtra struct {
	Name                string
	Namespace           string
	Replicas            int32
	AvailableReplicas   int32
	ReadyReplicas       int32
	UpdatedReplicas     int32
	UnavailableReplicas int32
	Generation          int64
	CreationTimestamp   string
	Labels              []string
	Annotations         []string
	Selector            []string
	Conditions          []string
}

func (p DeploymentExtra) Copy() Copyable {
	return DeploymentExtra{
		Name:                p.Name,
		Namespace:           p.Namespace,
		Replicas:            p.Replicas,
		AvailableReplicas:   p.AvailableReplicas,
		ReadyReplicas:       p.ReadyReplicas,
		UpdatedReplicas:     p.UpdatedReplicas,
		UnavailableReplicas: p.UnavailableReplicas,
		Generation:          p.Generation,
		CreationTimestamp:   p.CreationTimestamp,
		Labels:              misc.DeepCopyArray(p.Labels),
		Annotations:         misc.DeepCopyArray(p.Annotations),
		Selector:            misc.DeepCopyArray(p.Selector),
		Conditions:          misc.DeepCopyArray(p.Conditions),
	}
}

func newDeploymentExtra(dep *appsv1.Deployment) DeploymentExtra {
	labels := misc.RenderMapOfStrings("Labels", dep.Labels)
	annotations := misc.RenderMapOfStrings("Annotations", dep.Annotations)
	selector := misc.RenderMapOfStrings("Selector", dep.Spec.Selector.MatchLabels)

	var conditions []string
	for _, cond := range dep.Status.Conditions {
		conditions = append(conditions, fmt.Sprintf("%s=%s", cond.Type, cond.Status))
	}

	sort.Strings(conditions)

	return DeploymentExtra{
		Name:                dep.Name,
		Namespace:           dep.Namespace,
		Replicas:            *dep.Spec.Replicas,
		AvailableReplicas:   dep.Status.AvailableReplicas,
		ReadyReplicas:       dep.Status.ReadyReplicas,
		UpdatedReplicas:     dep.Status.UpdatedReplicas,
		UnavailableReplicas: dep.Status.UnavailableReplicas,
		Generation:          dep.Generation,
		CreationTimestamp:   dep.CreationTimestamp.Format(time.RFC3339),
		Labels:              labels,
		Annotations:         annotations,
		Selector:            selector,
		Conditions:          conditions,
	}
}

func renderDeploymentExtra(extra DeploymentExtra) []string {
	output := []string{
		fmt.Sprintf("Name: %s", extra.Name),
		fmt.Sprintf("Namespace: %s", extra.Namespace),
		fmt.Sprintf("Replicas: %d", extra.Replicas),
		fmt.Sprintf("Available Replicas: %d", extra.AvailableReplicas),
		fmt.Sprintf("Ready Replicas: %d", extra.ReadyReplicas),
		fmt.Sprintf("Updated Replicas: %d", extra.UpdatedReplicas),
		fmt.Sprintf("Unavailable Replicas: %d", extra.UnavailableReplicas),
		fmt.Sprintf("Generation: %d", extra.Generation),
		fmt.Sprintf("Creation Timestamp: %s", extra.CreationTimestamp),
	}
	output = append(output, extra.Labels...)
	output = append(output, extra.Annotations...)
	output = append(output, extra.Selector...)
	output = append(output, extra.Conditions...)
	return output
}

type DeploymentRenderer struct {
}

func (r DeploymentRenderer) Render(resource Resource, details bool) []string {
	extra := resource.Extra.(DeploymentExtra)

	if details {
		return renderDeploymentExtra(extra)
	}

	return []string{resource.Key()}
}

type DeploymentWatcher struct {
}

func (n DeploymentWatcher) Tick() {
}

func (n DeploymentWatcher) Kind() string {
	return "Deployment"
}

func (n DeploymentWatcher) Renderer() ResourceRenderer {
	return DeploymentRenderer{}
}

func (n DeploymentWatcher) convert(obj runtime.Object) *appsv1.Deployment {
	ret, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil
	}
	return ret
}

func (n DeploymentWatcher) ToResource(obj runtime.Object) Resource {
	d := n.convert(obj)
	extra := newDeploymentExtra(d)
	return NewK8sResource(n.Kind(), d, extra)
}

func watchForDeployments(watcher *K8sWatcher, k conn.KhronosConn, ns string) error {
	watchChan, err := k.Client.AppsV1().Deployments(ns).Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), DeploymentWatcher{})

	return nil

}
