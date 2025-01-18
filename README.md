![khronoscope](https://github.com/user-attachments/assets/ed78c414-98e6-400e-b1a1-a18e82baf189)

# Khronoscope: Rewind Your Kubernetes Cluster's History with VCR-Style Controls

Khronoscope is a tool inspired by k9s that allows you to inspect the state of your Kubernetes cluster and travel back in time to see its state at any point since you started the application using VCR like controls.  

This project is in it's VERY early stages, not in Alpha yet really but what it does do at the moment:

- Connects to your current Kubernetes cluster
- Supports the following resources: Namespaces, Nodes, Pods, ReplicaSets, DaemonSets, Deployments, and Services
- Controls
	- Up/Down - Move the resource selection up and down the tree
   	- Enter - Toggle folding the tree node you have selected
	- Alt Up/Alt Down - Move by a larger step value
	- Shift Up/Shift Down - Move the detail window up and down 
	- Left/Right - Go backwards or forwards in time
	- Esc - Jump back to current time
	- Tab - Switch view orientation
	- Ctrl-C - Exit

# Disclaimer

This project is in its early stages of development and is not yet recommended for production use. While adjustments have been made to minimize load on the control plane, performance under heavy load has not yet been tested.

# Example
<img width="1452" alt="Screenshot 2025-01-10 at 2 07 26 PM" src="https://github.com/user-attachments/assets/d4eeac64-b203-40ff-a668-631055b06639" />

[![Alternate Text](https://github.com/user-attachments/assets/d4eeac64-b203-40ff-a668-631055b06639)](https://github.com/user-attachments/assets/c4780bc3-1e28-40b8-bd8b-372e97a038a2 "Khronoscope Vidoe Demo showing VCR controls")

# Internals
This application create a connection to your kubernetes cluster and starts watching for resource changes.  It has a data structure called [TemporalMap](https://github.com/hoyle1974/khronoscope/blob/main/temporal_map.go) that is used to manage all the resource changes.  The structure allows you to add/update/delete values stored in it and then you can query the map for it's state at a specific point in time.  This let's the application show you the state of the cluster at any point since the start of the application.  

Along with standard resource information the application also collects metrics data for Pods and Nodes.  This is currently done with polling but needs to be switched to a watch as well.

I use [BubbleTea](https://github.com/charmbracelet/bubbletea) for rendering and [LipGloss](https://github.com/charmbracelet/lipgloss) for coloring, both great projects.

# Adding new k8s resource types

A good example of how to add a new resource type can be seen in [service.go](https://github.com/hoyle1974/khronoscope/blob/main/service.go).  You implement something like the following, replacing **{K8sResourceType}** with the resource type you are implementing.

```
type {K8sResourceType}Renderer struct {
}

func format{K8sResourceType}Details(t *corev1.{K8sResourceType}) []string {
	var result []string

	// Basic details
	result = append(result, fmt.Sprintf("Name:           %s", t.Name))

	return result
}

func (r {K8sResourceType}Renderer) Render(resource Resource, details bool) []string {
	if details {
		return format{K8sResourceType}Details(resource.Object.(*corev1.{K8sResourceType}))
	}

	return []string{resource.Key()}
}

type {K8sResourceType}Watcher struct {
}

func (n {K8sResourceType}Watcher) Tick() {
}

func (n {K8sResourceType}Watcher) Kind() string {
	return "{K8sResourceType}"
}

func (n *{K8sResourceType}Watcher) Renderer() ResourceRenderer {
	return nil
}

func (n {K8sResourceType}Watcher) convert(obj runtime.Object) *corev1.{K8sResourceType} {
	ret, ok := obj.(*corev1.{K8sResourceType})
	if !ok {
		return nil
	}
	return ret
}

func (n {K8sResourceType}Watcher) ToResource(obj runtime.Object) Resource {
	return NewK8sResource(n.Kind(), n.convert(obj), n.Renderer())
}

func watchFor{K8sResourceType}(watcher *K8sWatcher, k KhronosConn) {
	watchChan, err := k.client.CoreV1().{K8sResourceType}("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), {K8sResourceType}Watcher{})
}
```
In [main.go](https://github.com/hoyle1974/khronoscope/blob/main/main.go) you will need to make a call to watchFor**{K8sResourceType}**.  You will see examples of other resources being watched in main as well.


# Future

In the future I could see a variation of this application being run as a server and storing it's data long term in a database so that multiple users can connect to it and be able to look through cluster state for longer amounts of time.

I'd also like to support other resources than what is currently supported, possibly a plugin system could be used to support any custom resources users want to add.  Ones that I'd like to support out of the box include:
- StatefulSet
- ConfigMaps
- PersistantVolume
- Secrets
