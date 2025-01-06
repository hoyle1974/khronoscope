package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-logr/logr"
)

type KhronosConn struct {
	client kubernetes.Interface
	mc     *metrics.Clientset
}

func createClient(kubeconfigPath string) (KhronosConn, error) {
	var kubeconfig *rest.Config

	klog.SetLogger(logr.Logger{})

	if kubeconfigPath != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return KhronosConn{}, fmt.Errorf("unable to load kubeconfig from %s: %v", kubeconfigPath, err)
		}
		kubeconfig = config
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			return KhronosConn{}, fmt.Errorf("unable to load in-cluster config: %v", err)
		}
		kubeconfig = config
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return KhronosConn{}, fmt.Errorf("unable to create a client: %v", err)
	}

	mc, err := metrics.NewForConfig(kubeconfig)
	if err != nil {
		return KhronosConn{}, fmt.Errorf("unable to create a metric client: %v", err)
	}

	return KhronosConn{client: client, mc: mc}, nil
}

type ResourceWatcher interface {
	Init(client kubernetes.Interface)
	OnWatchEvent(watch.Event) bool
}

var watcher = NewWatcher()

func main() {
	fmt.Println("starting")
	client, err := createClient("/Users/jstrohm/.kube/config")
	if err != nil {
		panic(err)
	}
	fmt.Println("client created")

	watchForDeployments(watcher, client)
	watchForDaemonSet(watcher, client)
	watchForReplicaSet(watcher, client)
	watchForNodes(watcher, client)
	watchForService(watcher, client)
	watchForPods(watcher, client)
	watchForNamespaces(watcher, client)

	// for {
	// 	fmt.Println("-------------")
	// 	r := watcher.GetStateAtTime(time.Now(), "Pod", "kube-system")
	// 	for _, rr := range r {
	// 		fmt.Println(rr.Key(), rr.GetExtra())
	// 	}
	// 	time.Sleep(time.Second * 5)
	// }

	p := tea.NewProgram(
		newSimplePage("This app is under construction"),
	)

	watcher.OnChange(func() {
		p.Send(1)
	})

	if err := p.Start(); err != nil {
		panic(err)
	}

}

// MODEL DATA
var count = 0
var adjust = time.Duration(0)

type simplePage struct {
	text string
}

func newSimplePage(text string) simplePage {
	return simplePage{text: text}
}

func (s simplePage) Init() tea.Cmd { return nil }

// VIEW

func (s simplePage) View() string {
	b := strings.Builder{}

	count++
	b.WriteString(fmt.Sprintf("%d : %v - %v\n", count, adjust.Seconds(), watcher.GetLog()))

	snapshot := watcher.GetStateAtTime(time.Now().Add(adjust), "", "")

	// Get list of namespaces
	namespaces := []string{}
	for _, r := range snapshot {
		if r.Kind == "Namespace" {
			namespaces = append(namespaces, r.Name)
		}
	}
	namespaces = append(namespaces, "")
	sort.Strings(namespaces)

	resources := map[string]map[string][]Resource{}
	for _, r := range snapshot {
		if r.Kind == "Namespace" {
			continue
		}
		temp, ok := resources[r.Namespace]
		if !ok {
			temp = map[string][]Resource{}
		}
		temp[r.Kind] = append(temp[r.Kind], r)
		resources[r.Namespace] = temp
	}

	for _, namespace := range namespaces {
		b.WriteString(namespace + "\n")

		kinds := []string{}
		for kind, _ := range resources[namespace] {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)

		for _, kind := range kinds {
			b.WriteString(" |--" + kind + "\n")

			rs := []string{}
			for _, resource := range resources[namespace][kind] {
				rs = append(rs, resource.Name+resource.String())
			}
			sort.Strings(rs)
			for _, r := range rs {
				b.WriteString(" |   |--" + r + "\n")
			}
		}
	}

	return b.String()
}

// UPDATE

func (s simplePage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		switch msg.(tea.KeyMsg).String() {
		case "ctrl+c":
			return s, tea.Quit
		case "left":
			adjust -= time.Second
			return s, nil
		case "right":
			adjust += time.Second
			if adjust > 0 {
				adjust = 0
			}
			return s, nil
		case "enter":
			adjust = 0
			return s, nil
		}
	case int:
		return s, nil
	}

	return s, nil
}
