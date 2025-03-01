![khronoscope](https://github.com/user-attachments/assets/ed78c414-98e6-400e-b1a1-a18e82baf189)

# Khronoscope: Rewind Your Kubernetes Cluster's History with VCR-Style Controls

Khronoscope is a tool inspired by k9s that allows you to inspect the state of your Kubernetes cluster and travel back in time to see its state at any point since you started the application using VCR like controls.

This project is in it's VERY early stages, not in Alpha yet really but what it does do at the moment:

- Connects to your current Kubernetes cluster
- Supports the following resources: ConfigMap, DaemonSet, Deployments, Namespaces, Nodes, PersistantVolume, Pods, ReplicaSets, Secrets, and Services
- Controls - For the latest keybindings run `khronoscope -k`
  - '1' - Press this to show the details of a resource
  - '2' - Press this to show any collected logs of a resource
  - 'L' - Filter resources by those currently logging
  - 'l' - Toggle log collection for this pod
  - '/' - Filter by a string
  - 'q' - Save the recorded state to a file
  - 'm' - Mark this timestamp with a label
  - 'tab' - Rotate view
  - 'ctrl+c' - Quit
  - 'left' - In VCR mode, rewind, press multiple times to speed up
  - 'right' - In VCR mode, fast forward, press multiple times to speed up
  - ' ' - In VCR mode, toggle play/pause
  - 'esc' - Exit VCR mode and resume at the latest timestamp
  - 'enter' - Toggle folding the resource category/kind view
  - 'shift+up' - Jump up in the details view by 10 lines
  - 'shift+down' - Jump down in the details view by 10 lines
  - 'up' - Go up a resource
  - 'down' - Go down a resource
  - 'alt+up' - Go up 10 resources
  - 'alt+down' - Go down 10 resources
  - 'ctrl+d' - Delete a resource
  - 's' - Exec into a shell for this pod
  - 'P' - Filter all pods
  - 'ctrl+d' - Debug log window
  - 'shift+right' - In VCR mode, jump to the next marked label
  - 'shift+left' - In VCR mode, jump to the previous marked label
  
# Disclaimer

This project is in its early stages of development and is not yet recommended for production use. While adjustments have been made to minimize load on the control plane, performance under heavy load has not yet been tested.

# 📖 Debugging the Past with Khronoscope

Imagine inspecting your Kubernetes cluster when you notice something strange—a deployment with flapping pods. They start, crash, restart. Something’s off.

You pause the cluster state and check related resources. Nothing obvious. Rewinding a few minutes, you see the pods failing right after startup. Fast-forwarding, you mark one to start collecting logs. More crashes. Rewinding again, you inspect the logs just before failure—each pod dies trying to connect to a missing service.

Jumping to another namespace, you spot the issue: a critical infrastructure pod failed to start earlier. A quick fix, a restart, and everything stabilizes.

With Khronoscope’s ability to navigate through time, track key logs, and inspect past states, you solve in minutes what could’ve taken hours.

# Installation

Currently Khronoscope has only been tested on MacOS.  To install the latest build with homebrew:

```
brew tap hoyle1974/homebrew-tap
brew install khronoscope
```

You can also compile from source:

- Clone the repo
- Build and run the executable

```
https://github.com/hoyle1974/khronoscope.git
cd khronoscope
go run cmd/khronoscope/main.go
```

# Example
<img width="1452" alt="Screenshot 2025-01-10 at 2 07 26 PM" src="https://github.com/user-attachments/assets/d4eeac64-b203-40ff-a668-631055b06639" />

[![Alternate Text](https://github.com/user-attachments/assets/d4eeac64-b203-40ff-a668-631055b06639)](https://github.com/user-attachments/assets/c4780bc3-1e28-40b8-bd8b-372e97a038a2 "Khronoscope Vidoe Demo showing VCR controls")

# Contributions
I'm happy to have folks add contributions.  I've already created some [Issues](https://github.com/hoyle1974/khronoscope/labels/good%20first%20issue) that are great places to start if you want to contribute something useful but easy and self contained.  For more complex stuff feel free to add comments to the issues and I'm happy to discuss or create your own issues.  I'm really looking for help on how to make this tool more usable in real world scenarios, specifically in UI controls and added features!

# Internals
This application create a connection to your kubernetes cluster and starts watching for resource changes.  It has a data structure called [TemporalMap](https://github.com/hoyle1974/khronoscope/blob/main/internal/temporal/map.go) that is used to manage all the resource changes.  The structure allows you to add/update/delete values stored in it and then you can query the map for it's state at a specific point in time.  This let's the application show you the state of the cluster at any point since the start of the application.  The data in TemporalMap is actually stored as Keyframes and Diffs to reduce in memory pressure.  It's pretty efficient.

Along with standard resource information the application also collects metrics data for Pods and Nodes al.  Logs can be collected form pods as needed.  You can mark pods for log collection and then jump between them to inspect the logs at any point in time.

You can also mark timestamps with labels and jump between them in VCR mode.

I use [BubbleTea](https://github.com/charmbracelet/bubbletea) for rendering and [LipGloss](https://github.com/charmbracelet/lipgloss) for coloring, both great projects.

# Adding new k8s resource types

A good example of how to add a new resource type can be seen in [service.go](https://github.com/hoyle1974/khronoscope/blob/main/internal/resources/service.go).  To be honest a few of these I used ChatGPT to generate for me because it's so boilerplate and repetitive.  This is an example of the prompt I did.

```Rewrite this for k8s {new resource}} instead of service:```

and then I'd paste in ```service.go```, I'd then take the new code and add it, call gob.register on the new Extra type, and in watch.go I'd add it to the lists of watches to start.  I found this was a pretty good first pass.

# Future

In the future I could see a variation of this application being run as a server and storing it's data long term in a database so that multiple users can connect to it and be able to look through cluster state for longer amounts of time.

I'd also like to support other resources than what is currently supported, possibly a plugin system could be used to support any custom resources users want to add.
