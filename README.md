# Khronoscope
WARNING: You probably don't want to point this to a prod Kubernetes cluster yet.  I have made adjustments to make sure it doesn't overload the control plane but haven't tested it's load yet.

Imagine a tool like k9s that also lets you to inspect the state of your Kubernetes cluster BUT also let's you go back in time to see the state as it was at any point in time since you started the app.

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

# Example
<img width="1452" alt="Screenshot 2025-01-10 at 2 07 26 PM" src="https://github.com/user-attachments/assets/d4eeac64-b203-40ff-a668-631055b06639" />

# Internals
This application create a connection to your kubernetes cluster and starts watching for resource changes.  It has a data structure called [TemporalMap](https://github.com/hoyle1974/khronoscope/blob/main/temporal_map.go) that is used to manage all the resource changes.  The structure allows you to add/update/delete values stored in it and then you can query the map for it's state at a specific point in time.  This let's the application show you the state of the cluster at any point since the start of the application.  

Along with standard resource information the application also collects metrics data for Pods and Nodes.  This is currently done with polling but needs to be switched to a watch as well.

In the future I could see a variation of this application being run as a server and storing it's data long term in a database so that multiple users can connect to it and be able to look through cluster state for longer amounts of time.

I'd also like to support other resources than what is currently supported, possibly a plugin system could be used to support any custom resources users want to add.

I use [BubbleTea](https://github.com/charmbracelet/bubbletea) for rendering and [LipGloss](https://github.com/charmbracelet/lipgloss) for coloring, both great projects.
