# khronoscope
WARNING: You probably don't want to point this to a prod Kubernetes cluster yet.  I'm made adjustments to make sure it doesn't overload the control plane but haven't tested it's load yet.

Imagine a tool like k9s that also lets you to inspect the state of your Kubernetes cluster BUT also let's you go back in time to see the state as it was at any point in time since you started the app.

This project is in it's VERY early stages, not in Alpha yet really but what it does do at the moment:

- Connects to your current Kubernetes cluster
- Supports the following resources: Namespaces, Nodes, Pods, ReplicaSets, DaemonSets, Deployments, and Services
- Controls
	Up/Down - Move the resource selection up and down the tree
	Alt Up/Alt Down - Move by a larger step value
	Shift Up/Shift Down - Move the detail window up and down 
	Left/Right - Go backwards or forwards in time
	Esc - Jump back to current time
	Tap - Switch view orientation
	Ctrl-C - Exit

