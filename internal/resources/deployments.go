package resources

import (
	"context"
	"fmt"
	"sort"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/serializable"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DeploymentExtra holds the complete deployment state in a serializable form for GOB encoding
type DeploymentExtra struct {
	Replicas              int32
	UpdatedReplicas       int32
	ReadyReplicas         int32
	AvailableReplicas     int32
	UnavailableReplicas   int32
	StrategyType          string
	MinReadySeconds       int32
	RollingUpdateStrategy string
	Selector              []string
	Labels                []string
	Annotations           []string
	PodTemplate           PodTemplateExtra
	Conditions            []string
	Events                []string
	Age                   serializable.Time
}

// PodTemplateExtra holds the pod template details
type PodTemplateExtra struct {
	Labels            []string
	Annotations       []string
	ServiceAccount    string
	Containers        []ContainerExtra
	Volumes           []string
	PriorityClassName string
	NodeSelector      []string
	Tolerations       []string
}

// ContainerExtra holds the container details
type ContainerExtra struct {
	Name        string
	Image       string
	Ports       []string
	HostPorts   []string
	Args        []string
	Limits      []string
	Requests    []string
	Liveness    string
	Readiness   string
	Environment []string
	Mounts      []string
}

// Copy performs a deep copy of DeploymentExtra
func (p DeploymentExtra) Copy() Copyable {
	return DeploymentExtra{
		Replicas:              p.Replicas,
		UpdatedReplicas:       p.UpdatedReplicas,
		ReadyReplicas:         p.ReadyReplicas,
		AvailableReplicas:     p.AvailableReplicas,
		UnavailableReplicas:   p.UnavailableReplicas,
		StrategyType:          p.StrategyType,
		MinReadySeconds:       p.MinReadySeconds,
		RollingUpdateStrategy: p.RollingUpdateStrategy,
		Selector:              misc.DeepCopyArray(p.Selector),
		Labels:                misc.DeepCopyArray(p.Labels),
		Annotations:           misc.DeepCopyArray(p.Annotations),
		PodTemplate:           p.PodTemplate.Copy(),
		Conditions:            misc.DeepCopyArray(p.Conditions),
		Events:                misc.DeepCopyArray(p.Events),
		Age:                   p.Age,
	}
}

// Copy performs a deep copy of PodTemplateExtra
func (p PodTemplateExtra) Copy() PodTemplateExtra {
	return PodTemplateExtra{
		Labels:            misc.DeepCopyArray(p.Labels),
		Annotations:       misc.DeepCopyArray(p.Annotations),
		ServiceAccount:    p.ServiceAccount,
		Containers:        misc.DeepCopyArray(p.Containers),
		Volumes:           misc.DeepCopyArray(p.Volumes),
		PriorityClassName: p.PriorityClassName,
		NodeSelector:      misc.DeepCopyArray(p.NodeSelector),
		Tolerations:       misc.DeepCopyArray(p.Tolerations),
	}
}

// Copy performs a deep copy of ContainerExtra
func (c ContainerExtra) Copy() ContainerExtra {
	return ContainerExtra{
		Name:        c.Name,
		Image:       c.Image,
		Ports:       misc.DeepCopyArray(c.Ports),
		HostPorts:   misc.DeepCopyArray(c.HostPorts),
		Args:        misc.DeepCopyArray(c.Args),
		Limits:      misc.DeepCopyArray(c.Limits),
		Requests:    misc.DeepCopyArray(c.Requests),
		Liveness:    c.Liveness,
		Readiness:   c.Readiness,
		Environment: misc.DeepCopyArray(c.Environment),
		Mounts:      misc.DeepCopyArray(c.Mounts),
	}
}

// newDeploymentExtra constructs a DeploymentExtra from a Kubernetes Deployment
func newDeploymentExtra(deploy *appsv1.Deployment) DeploymentExtra {
	if deploy == nil {
		return DeploymentExtra{}
	}

	// Process selector with proper formatting
	selector := make([]string, 0, len(deploy.Spec.Selector.MatchLabels))
	for k, v := range deploy.Spec.Selector.MatchLabels {
		selector = append(selector, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(selector)

	// Process pod template
	podTemplate := PodTemplateExtra{
		Labels:            misc.RenderMapOfStrings(deploy.Spec.Template.Labels),
		Annotations:       misc.RenderMapOfStrings(deploy.Spec.Template.Annotations),
		ServiceAccount:    deploy.Spec.Template.Spec.ServiceAccountName,
		PriorityClassName: deploy.Spec.Template.Spec.PriorityClassName,
		NodeSelector:      misc.RenderMapOfStrings(deploy.Spec.Template.Spec.NodeSelector),
		Tolerations:       renderTolerations(deploy.Spec.Template.Spec.Tolerations),
	}

	// Process containers
	for _, container := range deploy.Spec.Template.Spec.Containers {
		containerExtra := ContainerExtra{
			Name:        container.Name,
			Image:       container.Image,
			Ports:       renderContainerPorts(container.Ports),
			HostPorts:   renderContainerPorts(container.Ports),
			Args:        container.Args,
			Limits:      renderResourceList(container.Resources.Limits),
			Requests:    renderResourceList(container.Resources.Requests),
			Liveness:    renderProbe(container.LivenessProbe),
			Readiness:   renderProbe(container.ReadinessProbe),
			Environment: renderEnvVars(container.Env),
			Mounts:      renderVolumeMounts(container.VolumeMounts),
		}
		podTemplate.Containers = append(podTemplate.Containers, containerExtra)
	}

	// Process volumes
	for _, volume := range deploy.Spec.Template.Spec.Volumes {
		volumeStr := fmt.Sprintf("%s: %s", volume.Name, renderVolumeSource(volume.VolumeSource))
		podTemplate.Volumes = append(podTemplate.Volumes, volumeStr)
	}

	// Process conditions
	conditions := make([]string, 0, len(deploy.Status.Conditions))
	for _, condition := range deploy.Status.Conditions {
		conditions = append(conditions, fmt.Sprintf("%s: %s", condition.Type, condition.Status))
	}

	return DeploymentExtra{
		Replicas:              *deploy.Spec.Replicas,
		UpdatedReplicas:       deploy.Status.UpdatedReplicas,
		ReadyReplicas:         deploy.Status.ReadyReplicas,
		AvailableReplicas:     deploy.Status.AvailableReplicas,
		UnavailableReplicas:   deploy.Status.UnavailableReplicas,
		StrategyType:          string(deploy.Spec.Strategy.Type),
		MinReadySeconds:       deploy.Spec.MinReadySeconds,
		RollingUpdateStrategy: renderRollingUpdateStrategy(deploy.Spec.Strategy.RollingUpdate),
		Selector:              selector,
		Labels:                misc.RenderMapOfStrings(deploy.Labels),
		Annotations:           misc.RenderMapOfStrings(deploy.Annotations),
		PodTemplate:           podTemplate,
		Conditions:            conditions,
		Age:                   serializable.NewTime(deploy.CreationTimestamp.Time),
	}
}

// renderRollingUpdateStrategy renders the rolling update strategy
func renderRollingUpdateStrategy(rollingUpdate *appsv1.RollingUpdateDeployment) string {
	if rollingUpdate == nil {
		return "<none>"
	}
	maxSurge := "<unknown>"
	if rollingUpdate.MaxSurge != nil {

		if rollingUpdate.MaxSurge.Type == intstr.Int {
			maxSurge = fmt.Sprintf("%d", rollingUpdate.MaxSurge.IntVal)
		} else {
			maxSurge = fmt.Sprintf("%d", rollingUpdate.MaxSurge.IntVal)
		}
	}

	return fmt.Sprintf("%d max unavailable, %s%% max surge",
		rollingUpdate.MaxUnavailable.IntVal,
		maxSurge)
}

// renderContainerPorts renders container ports
func renderContainerPorts(ports []corev1.ContainerPort) []string {
	result := make([]string, 0, len(ports))
	for _, port := range ports {
		result = append(result, fmt.Sprintf("%d/%s", port.ContainerPort, port.Protocol))
	}
	return result
}

// renderResourceList renders resource limits and requests
func renderResourceList(resources corev1.ResourceList) []string {
	result := make([]string, 0, len(resources))
	for k, v := range resources {
		result = append(result, fmt.Sprintf("%s: %s", k, v.String()))
	}
	return result
}

// renderProbe renders a probe
func renderProbe(probe *corev1.Probe) string {
	if probe == nil {
		return "<none>"
	}
	return fmt.Sprintf("%s", probe.ProbeHandler)
}

// renderEnvVars renders environment variables
func renderEnvVars(env []corev1.EnvVar) []string {
	result := make([]string, 0, len(env))
	for _, e := range env {
		result = append(result, fmt.Sprintf("%s: %s", e.Name, e.Value))
	}
	return result
}

// renderVolumeMounts renders volume mounts
func renderVolumeMounts(mounts []corev1.VolumeMount) []string {
	result := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		result = append(result, fmt.Sprintf("%s: %s", mount.Name, mount.MountPath))
	}
	return result
}

// renderVolumeSource renders volume source
func renderVolumeSource(source corev1.VolumeSource) string {
	if source.ConfigMap != nil {
		return fmt.Sprintf("ConfigMap: %s", source.ConfigMap.Name)
	}
	if source.Secret != nil {
		return fmt.Sprintf("Secret: %s", source.Secret.SecretName)
	}
	return "<none>"
}

// renderTolerations renders tolerations
func renderTolerations(tolerations []corev1.Toleration) []string {
	result := make([]string, 0, len(tolerations))
	for _, tol := range tolerations {
		result = append(result, fmt.Sprintf("%s: %s", tol.Key, tol.Value))
	}
	return result
}

type DeploymentRenderer struct{}

func renderDeploymentExtra(extra DeploymentExtra) []string {
	output := []string{
		fmt.Sprintf("Replicas:                %d desired | %d updated | %d total | %d available | %d unavailable", extra.Replicas, extra.UpdatedReplicas, extra.Replicas, extra.AvailableReplicas, extra.UnavailableReplicas),
		fmt.Sprintf("StrategyType:            %s", extra.StrategyType),
		fmt.Sprintf("MinReadySeconds:         %d", extra.MinReadySeconds),
		fmt.Sprintf("RollingUpdateStrategy:   %s", extra.RollingUpdateStrategy),
		fmt.Sprintf("Selector:                %s", misc.FormatNilArray(extra.Selector)),
	}

	// Add Labels section if present
	if len(extra.Labels) > 0 {
		output = append(output, "Labels:")
		for _, label := range extra.Labels {
			output = append(output, fmt.Sprintf("                          %s", label))
		}
	}

	// Add Annotations section if present
	if len(extra.Annotations) > 0 {
		output = append(output, "Annotations:")
		for _, annotation := range extra.Annotations {
			output = append(output, fmt.Sprintf("                          %s", annotation))
		}
	}

	// Add Pod Template section
	output = append(output, "Pod Template:")
	output = append(output, fmt.Sprintf("  Labels:                %s", misc.FormatNilArray(extra.PodTemplate.Labels)))
	output = append(output, fmt.Sprintf("  Annotations:           %s", misc.FormatNilArray(extra.PodTemplate.Annotations)))
	output = append(output, fmt.Sprintf("  Service Account:       %s", extra.PodTemplate.ServiceAccount))
	output = append(output, fmt.Sprintf("  Priority Class Name:   %s", extra.PodTemplate.PriorityClassName))
	output = append(output, fmt.Sprintf("  Node Selector:         %s", misc.FormatNilArray(extra.PodTemplate.NodeSelector)))
	output = append(output, fmt.Sprintf("  Tolerations:           %s", misc.FormatNilArray(extra.PodTemplate.Tolerations)))

	// Add Containers section
	for _, container := range extra.PodTemplate.Containers {
		output = append(output, fmt.Sprintf("  Container:             %s", container.Name))
		output = append(output, fmt.Sprintf("    Image:               %s", container.Image))
		output = append(output, fmt.Sprintf("    Ports:               %s", misc.FormatNilArray(container.Ports)))
		output = append(output, fmt.Sprintf("    Host Ports:          %s", misc.FormatNilArray(container.HostPorts)))
		output = append(output, fmt.Sprintf("    Args:                %s", misc.FormatNilArray(container.Args)))
		output = append(output, fmt.Sprintf("    Limits:              %s", misc.FormatNilArray(container.Limits)))
		output = append(output, fmt.Sprintf("    Requests:            %s", misc.FormatNilArray(container.Requests)))
		output = append(output, fmt.Sprintf("    Liveness:            %s", container.Liveness))
		output = append(output, fmt.Sprintf("    Readiness:           %s", container.Readiness))
		output = append(output, fmt.Sprintf("    Environment:         %s", misc.FormatNilArray(container.Environment)))
		output = append(output, fmt.Sprintf("    Mounts:              %s", misc.FormatNilArray(container.Mounts)))
	}

	// Add Volumes section
	output = append(output, fmt.Sprintf("  Volumes:               %s", misc.FormatNilArray(extra.PodTemplate.Volumes)))

	// Add Conditions section
	output = append(output, fmt.Sprintf("Conditions:              %s", misc.FormatNilArray(extra.Conditions)))

	return output
}

func (r DeploymentRenderer) Render(resource Resource, details bool) []string {
	extra, ok := resource.Extra.(DeploymentExtra)
	if !ok {
		return []string{"Error: Invalid extra type"}
	}

	if details {
		return renderDeploymentExtra(extra)
	}
	return []string{resource.Key()}
}

type DeploymentWatcher struct{}

func (n DeploymentWatcher) Tick()                      {}
func (n DeploymentWatcher) Kind() string               { return "Deployment" }
func (n DeploymentWatcher) Renderer() ResourceRenderer { return DeploymentRenderer{} }

func (n DeploymentWatcher) convert(obj runtime.Object) *appsv1.Deployment {
	if obj == nil {
		return nil
	}
	ret, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil
	}
	return ret
}

func (n DeploymentWatcher) ToResource(obj runtime.Object) Resource {
	deploy := n.convert(obj)
	if deploy == nil {
		return Resource{}
	}
	extra := newDeploymentExtra(deploy)
	return NewK8sResource(n.Kind(), deploy, extra)
}

func watchForDeployment(watcher *K8sWatcher, k conn.KhronosConn, ns string) error {
	watchChan, err := k.Client.AppsV1().Deployments(ns).Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to watch deployments: %w", err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), DeploymentWatcher{})
	return nil
}
