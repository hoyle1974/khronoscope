package main

import (
	"fmt"
	"os"
	"runtime/pprof"
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
	ready             bool
	viewMode          int
	width             int
	height            int
	treeView          viewport.Model
	detailView        viewport.Model
	lastWindowSizeMsg tea.WindowSizeMsg
	tv                *TreeView
}

func newSimplePage() *simplePage {
	return &simplePage{
		tv: NewTreeView(),
	}
}

func (m *simplePage) headerView() string {
	title := titleStyle.Render("Khronoscope")
	line := strings.Repeat("─", max(0, m.width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m *simplePage) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.treeView.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (s *simplePage) Init() tea.Cmd { return nil }

// VIEW

var curPosition = 0
var curRealPosition = 0
var count = 0

func (m *simplePage) View() string {
	m.tv.AddResources(watcher.GetStateAtTime(time.Now().Add(adjust), "", ""))

	treeContent, focusLine, resource := m.tv.Render()
	m.treeView.SetContent(treeContent)
	m.treeView.YOffset = focusLine - (m.treeView.Height / 2)
	if m.treeView.YOffset < 0 {
		m.treeView.YOffset = 0
	}

	if resource != nil {
		m.detailView.SetContent(fmt.Sprintf("UID: %s\n", resource.Uid) + strings.Join(resource.Details(), "\n"))
	}

	fixWidth := func(s string, width int) string {
		ss := strings.Split(s, "\n")
		if len(ss) > 0 {
			// Calculate the number of spaces needed
			padding := width - len(ss[0])
			if padding > 0 {
				// Append spaces to the string
				ss[0] = ss[0] + strings.Repeat(" ", padding)
			}
		}
		return strings.Join(ss, "\n")
	}

	temp := ""
	if m.viewMode == 0 {
		temp = lipgloss.JoinHorizontal(0, fixWidth(m.treeView.View(), m.width/2), " ", m.detailView.View())
	} else {
		temp = lipgloss.JoinVertical(0, fixWidth(m.treeView.View(), m.width), m.detailView.View())
	}

	// log := fmt.Sprintf("%d : %v - %v\n", count, adjust.Seconds(), watcher.GetLog())

	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), temp, m.footerView())
}

// UPDATE
func (m *simplePage) windowResize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height

	m.lastWindowSizeMsg = msg
	headerHeight := lipgloss.Height(m.headerView())
	footerHeight := lipgloss.Height(m.footerView())
	verticalMarginHeight := headerHeight + footerHeight

	updateViews := func() {
		if m.viewMode == 0 {
			m.treeView.Width = msg.Width / 2
			m.treeView.Height = msg.Height - verticalMarginHeight

			m.detailView.Width = msg.Width / 2
			m.detailView.Height = msg.Height - verticalMarginHeight
		} else {
			m.treeView.Width = msg.Width
			m.treeView.Height = msg.Height/2 - verticalMarginHeight

			m.detailView.Width = msg.Width
			m.detailView.Height = msg.Height/2 - verticalMarginHeight
		}
	}

	if !m.ready {
		// Since this program is using the full size of the viewport we
		// need to wait until we've received the window dimensions before
		// we can initialize the viewport. The initial dimensions come in
		// quickly, though asynchronously, which is why we wait for them
		// here.
		m.treeView = viewport.New(msg.Width/2, msg.Height-verticalMarginHeight)
		m.treeView.YPosition = headerHeight
		m.treeView.HighPerformanceRendering = useHighPerformanceRenderer

		m.detailView = viewport.New(msg.Width/2, msg.Height-verticalMarginHeight)
		m.detailView.YPosition = headerHeight
		m.detailView.HighPerformanceRendering = useHighPerformanceRenderer

		m.ready = true

		updateViews()

		// This is only necessary for high performance rendering, which in
		// most cases you won't need.
		//
		// Render the viewport one line below the header.
		m.treeView.YPosition = headerHeight + 1
	} else {
		updateViews()
	}
}

func (m *simplePage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.viewMode++
			m.viewMode %= 2
			m.windowResize(m.lastWindowSizeMsg)
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		case "left":
			adjust -= time.Second
			return m, nil
		case "right":
			adjust += time.Second
			if adjust > 0 {
				adjust = 0
			}
			return m, nil
		case "enter":
			adjust = 0
			m.tv.Toggle()
			return m, nil
		case "up":
			m.tv.Up()
			return m, nil
		case "down":
			m.tv.Down()
			return m, nil
		case "alt+up":
			m.tv.PageUp()
			return m, nil
		case "alt+down":
			m.tv.PageDown()
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.windowResize(msg)

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(m.treeView))
		}

	case int:
		m.treeView, cmd = m.treeView.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}

	// Handle keyboard and mouse events in the viewport
	// s.viewport, cmd = s.viewport.Update(msg)
	// cmds = append(cmds, cmd)

	// return s, tea.Batch(cmds...)
	return m, nil
}
