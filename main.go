package main

import (
	"fmt"
	"os"
	"runtime/pprof"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/go-logr/logr"
)

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()

	bold = lipgloss.NewStyle().Bold(true)
)

const useHighPerformanceRenderer = false

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

	// Start profiling
	profile := false
	if profile {
		f, err := os.Create("khronoscope.prof")
		if err != nil {
			fmt.Println(err)
			return

		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	p := tea.NewProgram(
		newSimplePage(),
	)

	watcher.OnChange(func() {
		p.Send(1)
	})

	if err := p.Start(); err != nil {
		panic(err)
	}

}

// MODEL DATA
var adjust = time.Duration(0)

type simplePage struct {
	ready    bool
	viewport viewport.Model
}

func newSimplePage() *simplePage {
	return &simplePage{}
}

func (m *simplePage) headerView() string {
	title := titleStyle.Render("Khronoscope")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m *simplePage) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (s *simplePage) Init() tea.Cmd { return nil }

// VIEW

func grommet(is bool) string {
	if !is {
		return "├"
	}
	return "└"
}
func grommet2(is bool) string {
	if !is {
		return "│"
	}
	return " "
}

var curPosition = 0

func (s *simplePage) View() string {

	b := strings.Builder{}

	// count++
	// b.WriteString(fmt.Sprintf("%d : %v - %v\n", count, adjust.Seconds(), watcher.GetLog()))

	snapshot := watcher.GetStateAtTime(time.Now().Add(adjust), "", "")

	// Namespaces
	namespaces := []string{}
	for _, r := range snapshot {
		if r.Kind == "Namespace" {
			namespaces = append(namespaces, r.Name)
		}
	}
	namespaces = append(namespaces, "")
	sort.Strings(namespaces)

	// Map of resources by namespace/kind
	resources := map[string]map[string][]Resource{}
	for _, r := range snapshot {
		temp, ok := resources[r.Namespace]
		if !ok {
			temp = map[string][]Resource{}
		}
		temp[r.Kind] = append(temp[r.Kind], r)
		resources[r.Namespace] = temp
	}

	pos := -1
	selected := false
	details := ""

	mark := func() string {
		pos++
		if curPosition == pos {
			selected = true
			return "[*] "
		}
		selected = false
		return "[ ] "
	}

	unmark := func() string {
		return "    "
	}

	// Nodes & Namespaces
	b.WriteString("\n")
	for _, namespace := range namespaces {
		if len(namespace) != 0 {
			continue // skip things that are not nodes
		}
		kinds := []string{}
		for kind, _ := range resources[namespace] {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)

		for _, kind := range kinds {
			b.WriteString(bold.Render(kind) + "\n")

			rs := []Resource{}
			rs = append(rs, resources[namespace][kind]...)
			sort.Slice(rs, func(i, j int) bool {
				return rs[i].Name < rs[j].Name
			})

			for idx, r := range rs {
				render := r.String()
				if len(render) == 0 {
					b.WriteString(mark() + " " + grommet(idx == len(rs)-1) + "──" + r.Name + "\n")
					if selected {
						details = strings.Join(r.Details(), "\n")
					}
				} else {
					for idx2, s := range render {
						if idx2 == 0 {
							b.WriteString(mark() + " " + grommet(idx == len(rs)-1) + "──" + r.Name + s + "\n")
							if selected {
								details = strings.Join(r.Details(), "\n")
							}
						} else {
							b.WriteString(unmark() + " │  " + s + "\n")
						}
					}
				}
			}
		}
	}

	// All namespaced resources
	b.WriteString("\n")
	for _, namespace := range namespaces {
		if len(namespace) == 0 {
			continue // skip nodes
		}
		b.WriteString(bold.Render(namespace) + "\n")

		kinds := []string{}
		for kind, _ := range resources[namespace] {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)

		for idx, kind := range kinds {
			b.WriteString(unmark() + " " + grommet(idx == len(kinds)-1) + "──" + bold.Render(kind) + "\n")

			rs := []Resource{}
			rs = append(rs, resources[namespace][kind]...)
			sort.Slice(rs, func(i, j int) bool {
				return rs[i].Name < rs[j].Name
			})

			for idx2, r := range rs {
				render := r.String()
				if len(render) == 0 {
					b.WriteString(mark() + " │   " + grommet(idx2 == len(rs)-1) + "──" + r.Name + "\n")
					if selected {
						details = strings.Join(r.Details(), "\n")
					}
				} else {
					for idx3, s := range render {
						if idx3 == 0 {
							b.WriteString(mark() + " │   " + grommet(idx2 == len(rs)-1) + "──" + r.Name + s + "\n")
							if selected {
								details = strings.Join(r.Details(), "\n")
							}
						} else {
							b.WriteString(unmark() + " │   " + grommet2(idx2 == len(rs)-1) + "  " + s + "\n")
						}
					}
				}
			}
		}
	}

	content := b.String()

	if curPosition < 0 {
		curPosition = 0
	} else if curPosition > pos {
		curPosition = pos
	}

	s.viewport.SetContent(content)

	if !s.ready {
		return "\n  Initializing..."
	}

	ds := lipgloss.NewStyle().Width(60).Height(s.viewport.Height)

	temp := lipgloss.JoinHorizontal(0, s.viewport.View(), ds.Render(details))

	return fmt.Sprintf("%s\n%s\n%s", s.headerView(), temp, s.footerView())
}

// UPDATE

func (s *simplePage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
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
		case "up":
			curPosition--
			s.viewport.LineUp(1)
			//begin := s.viewport.YOffset
			//end := s.viewport.YOffset + s.viewport.Height

			return s, nil
		case "down":
			curPosition++
			s.viewport.LineDown(1)
			return s, nil
		}
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(s.headerView())
		footerHeight := lipgloss.Height(s.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !s.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			s.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			s.viewport.YPosition = headerHeight
			s.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			s.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			s.viewport.YPosition = headerHeight + 1
		} else {
			s.viewport.Width = msg.Width
			s.viewport.Height = msg.Height - verticalMarginHeight
		}

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(s.viewport))
		}

	case int:
		s.viewport, cmd = s.viewport.Update(msg)
		cmds = append(cmds, cmd)

		return s, tea.Batch(cmds...)
	}

	// Handle keyboard and mouse events in the viewport
	// s.viewport, cmd = s.viewport.Update(msg)
	// cmds = append(cmds, cmd)

	// return s, tea.Batch(cmds...)
	return s, nil
}
