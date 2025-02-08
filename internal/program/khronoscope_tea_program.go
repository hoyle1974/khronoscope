package program

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hoyle1974/khronoscope/internal/config"
	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/dao"
	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/hoyle1974/khronoscope/internal/types"
	"github.com/hoyle1974/khronoscope/internal/ui"
)

type KhronoscopeTeaProgram struct {
	data              dao.KhronoStore
	watcher           *resources.K8sWatcher
	ready             bool
	viewMode          int
	width             int
	height            int
	treeView          viewport.Model
	detailView        viewport.Model
	lastWindowSizeMsg tea.WindowSizeMsg
	tv                *ui.TreeController
	VCR               *ui.PlaybackController
	popup             ui.Popup
	search            bool
	searchFilter      ui.Filter
	searchInput       textinput.Model
	logCollector      *resources.LogCollector
	tab               int
	cfg               config.Config
	client            conn.KhronosConn
}

func (m *KhronoscopeTeaProgram) SetLabel(label string) {
	m.data.SetLabel(m.VCR.GetTimeToUse(), label)
}

func calculatePercentageOfTime(min, max, value time.Time) float64 {
	// Ensure the value is within the range
	if value.Before(min) || value.After(max) {
		return 0
	}

	// Convert to Unix timestamps or durations
	minUnix := min.Unix()
	maxUnix := max.Unix()
	valueUnix := value.Unix()

	// Calculate percentage
	percentage := float64(valueUnix-minUnix) / float64(maxUnix-minUnix)
	return percentage
}

func (m *KhronoscopeTeaProgram) SetPopup(popup ui.Popup) {
	m.popup = popup
}

func (m *KhronoscopeTeaProgram) headerView(label string) string {
	minTime, maxTime := m.data.GetTimeRange()
	current := m.VCR.GetTimeToUse()
	p := calculatePercentageOfTime(minTime, maxTime, current)

	currentTime := fmt.Sprintf(" Current Time: %s ", current.Format("2006-01-02 15:04:05"))

	percentText := fmt.Sprintf("Available Range (%s to %s) %3.2f%% ",
		minTime.Format("2006-01-02 15:04:05"),
		maxTime.Format("2006-01-02 15:04:05"),
		p*100,
	)
	bar := ""

	if !m.VCR.IsEnabled() {
		bar = currentTime
	} else {

		size := len(percentText)
		filledSegments := int(math.Round(p * float64(size)))

		// Define styles for filled and empty segments
		filledStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("#000000")) // Green
		emptyStyle := lipgloss.NewStyle().Background(lipgloss.Color("#0000FF")).Foreground(lipgloss.Color("#FFFFFF"))  // Gray

		// Build the bar
		bar = currentTime + " ["
		for i := 0; i < len(percentText); i++ {
			if i < filledSegments {
				bar += filledStyle.Render(string(percentText[i]))
			} else {
				bar += emptyStyle.Render(string(percentText[i]))
			}
		}
		bar += "]"
	}

	vcrStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FFAA00")).Foreground(lipgloss.Color("#000000"))

	if len(label) > 0 {
		label = "[" + label + "]"
	}
	if m.searchFilter != nil {
		label += " " + m.searchFilter.Description()
	}

	title := lipgloss.NewStyle().Render(fmt.Sprintf("Khronoscope %s - %s %s ",
		label,
		vcrStyle.Render("  "+m.VCR.Render()+"  "),
		bar,
	))
	line := strings.Repeat("─", max(0, m.width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m *KhronoscopeTeaProgram) footerView() string {
	info := lipgloss.NewStyle().Render(fmt.Sprintf(" %3.f%%", m.treeView.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func NewProgram(watcher *resources.K8sWatcher, d dao.KhronoStore, l *resources.LogCollector, client conn.KhronosConn) *KhronoscopeTeaProgram {
	am := &KhronoscopeTeaProgram{
		watcher:      watcher,
		data:         d,
		tv:           ui.NewTreeView(),
		logCollector: l,
		cfg:          config.Get(),
		client:       client,
	}

	return am
}

func (s *KhronoscopeTeaProgram) Init() tea.Cmd { return nil }

func (m *KhronoscopeTeaProgram) View() string {
	timeToUse := m.VCR.GetTimeToUse()
	resourcesNow := m.data.GetResourcesAt(timeToUse, "", "")
	convResources := make([]types.Resource, len(resourcesNow))
	for i := 0; i < len(resourcesNow); i++ {
		convResources[i] = resourcesNow[i]
	}
	m.tv.UpdateResources(convResources)

	currentLabel := m.data.GetLabel(timeToUse)

	m.tv.SetFilter(m.searchFilter)
	treeContent, focusLine := m.tv.Render(m.VCR.IsEnabled())
	treeContent = lipgloss.NewStyle().Width(m.treeView.Width).Render(treeContent)
	m.treeView.SetContent(treeContent)
	m.treeView.YOffset = focusLine - (m.treeView.Height / 2)
	if m.treeView.YOffset < 0 {
		m.treeView.YOffset = 0
	}

	resource := m.tv.GetSelected()
	if resource != nil {
		if m.tab == 1 && resource.GetKind() == "Pod" {
			logs := resource.(resources.Resource).Extra.(resources.PodExtra).Logs
			m.detailView.SetContent(strings.Join(logs, "\n"))
		} else {
			detailContent := fmt.Sprintf("UID: %s\n", resource.GetUID()) + strings.Join(resource.GetDetails(), "\n")
			detailContent = lipgloss.NewStyle().Width(m.detailView.Width).Render(detailContent)
			m.detailView.SetContent(detailContent)
		}

	} else {
		m.detailView.SetContent("")
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
		temp = lipgloss.JoinVertical(0, fixWidth(m.treeView.View(), m.width), " ", m.detailView.View())
	}

	var top string
	if m.search {
		top = m.searchInput.View()
	} else {
		top = m.headerView(currentLabel)
	}

	return m.insertPopup(fmt.Sprintf("%s\n%s\n%s", top, temp, m.footerView()), m.popup)
}

func (m *KhronoscopeTeaProgram) insertPopup(content string, popup ui.Popup) string {
	if popup == nil {
		return content
	}

	popupLines := strings.Split(popup.View(m.width, m.height), "\n")
	contentLines := strings.Split(content, "\n")

	ll := len(contentLines)
	for idx := 0; idx < m.height-ll; idx++ {
		contentLines = append(contentLines, "")
	}

	offset := (m.height / 2) - (len(popupLines) / 2)

	for idx, line := range popupLines {
		contentLines[idx+offset] = line
	}

	return strings.Join(contentLines, "\n")
}

// UPDATE
func (m *KhronoscopeTeaProgram) windowResize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height

	m.lastWindowSizeMsg = msg
	headerHeight := lipgloss.Height(m.headerView(""))
	footerHeight := lipgloss.Height(m.footerView())
	verticalMarginHeight := headerHeight + footerHeight

	updateViews := func() {
		if m.viewMode == 0 {
			m.treeView.Width = msg.Width / 2
			m.treeView.Height = msg.Height - verticalMarginHeight

			m.detailView.Width = (msg.Width / 2) - 1
			m.detailView.Height = msg.Height - verticalMarginHeight
		} else {
			m.treeView.Width = msg.Width
			m.treeView.Height = msg.Height/2 - verticalMarginHeight

			m.detailView.Width = msg.Width - 1
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

		m.detailView = viewport.New(msg.Width/2, msg.Height-verticalMarginHeight)
		m.detailView.YPosition = headerHeight

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

type searchFilter struct {
	value string
}

func (f searchFilter) Matches(r types.Resource) bool { return strings.Contains(r.String(), f.value) }
func (f searchFilter) Description() string           { return f.value }
func (f searchFilter) Highlight() string             { return f.value }

type podFilter struct {
}

func (f podFilter) Matches(r types.Resource) bool { return r.GetKind() == "Pod" }
func (f podFilter) Description() string           { return "Pods" }
func (f podFilter) Highlight() string             { return "" }

type logFilter struct {
}

func (f logFilter) Matches(r types.Resource) bool {
	if r.GetKind() != "Pod" {
		return false
	}
	if r.GetExtra() != nil {
		if pe, ok := r.GetExtra().(resources.PodExtra); ok {
			return len(pe.Logging) > 0
		}
	}
	return false
}

func (f logFilter) Description() string { return "Logging" }
func (f logFilter) Highlight() string   { return "" }

func (m *KhronoscopeTeaProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if m.popup != nil {
		if m.popup.Update(msg) {
			m.SetPopup(nil)
		}
		return m, nil
	}

	if m.search {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyEsc:
				m.search = false
				return m, nil
			case tea.KeyEnter:
				// Save the label
				if m.searchInput.Value() == "" {
					m.searchFilter = nil
				} else {
					m.searchFilter = searchFilter{m.searchInput.Value()}
				}

				m.search = false
				return m, nil
			}
		}

		m.searchInput, _ = m.searchInput.Update(msg)
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case m.cfg.KeyBindings.WindowDetails: //"1":
			m.tab = 0
			return m, nil
		case m.cfg.KeyBindings.WindowLogs: // "2":
			m.tab = 1
			return m, nil
		case m.cfg.KeyBindings.FilterLogsToggle: // "L":
			if m.searchFilter == nil {
				m.searchFilter = logFilter{}
			} else {
				m.searchFilter = nil
			}
			return m, nil
		case m.cfg.KeyBindings.Pod:
			if m.searchFilter == nil {
				m.searchFilter = podFilter{}
			} else {
				m.searchFilter = nil
			}
			return m, nil
		case m.cfg.KeyBindings.LogToggle: //:"l":
			if m.VCR.IsEnabled() {
				return m, nil // Can't toggle logs while in VCR mode
			}
			if sel := m.tv.GetSelected(); sel != nil {
				m.popup = ui.NewContainerPopupModel(m.client, sel, func(name string) {
					if name == "" {
						return
					}
					// Container was selected
					resources.ToggleLogs(sel, name)
				})
			}
			return m, nil
		case m.cfg.KeyBindings.FilterSearch: // "/":
			m.searchInput = textinput.New()
			m.searchInput.Placeholder = ""
			m.searchInput.Focus()
			m.searchInput.CharLimit = 156
			m.searchInput.Width = 20
			m.search = true
			return m, nil
		case m.cfg.KeyBindings.Save: // "s":
			m.data.Save("temp.dat")
			return m, nil
		case m.cfg.KeyBindings.Exec:
			if sel := m.tv.GetSelected(); sel != nil && sel.GetKind() == "Pod" {
				m.SetPopup(ui.NewExecPopupModel(m.client, sel))
			}
			return m, nil
		case m.cfg.KeyBindings.NewLabel: //"m":
			m.VCR.Pause()
			m.SetPopup(ui.NewLabelPopup(m))
			return m, nil
		case m.cfg.KeyBindings.RotateViewToggle: //"tab":
			m.viewMode++
			m.viewMode %= 2
			m.windowResize(m.lastWindowSizeMsg)
			return m, nil
		case m.cfg.KeyBindings.Quit: //"ctrl+c":
			return m, tea.Quit
		case m.cfg.KeyBindings.VCRRewind: //"left":
			m.VCR.Rewind()
			return m, nil
		case m.cfg.KeyBindings.VCRFastForward: // "right":
			m.VCR.FastForward()
			return m, nil
		case m.cfg.KeyBindings.VCRPlay: //" ":
			if m.VCR.GetPlaySpeed() == 0 {
				m.VCR.Play()
			} else {
				m.VCR.Pause()
			}
			return m, nil
		case m.cfg.KeyBindings.VCROff: // m.cfg.KeyBindings.Exit: // "esc":
			m.VCR.DisableVirtualTime()
		case m.cfg.KeyBindings.Toggle: // "enter":
			m.tv.Toggle()
			return m, nil
		case m.cfg.KeyBindings.DetailsUp: //"shift+up":
			m.detailView.LineUp(10)
			return m, nil
		case m.cfg.KeyBindings.DetailsDown: //"shift+down":
			m.detailView.LineDown(10)
			return m, nil
		case m.cfg.KeyBindings.Up: // "up":
			m.tv.Up()
			return m, nil
		case m.cfg.KeyBindings.Down: // "down":
			m.tv.Down()
			return m, nil
		case m.cfg.KeyBindings.PageUp: // "alt+up":
			m.tv.PageUp()
			return m, nil
		case m.cfg.KeyBindings.PageDown: // "alt+down":
			m.tv.PageDown()
			return m, nil
		case m.cfg.KeyBindings.DeleteResource:
			r := m.tv.GetSelected()
			if r != nil && r.GetKind() == "Pod" {
				_ = m.client.Client.CoreV1().Pods(r.GetNamespace()).Delete(context.Background(), r.GetName(), v1.DeleteOptions{})
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.windowResize(msg)
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
