package program

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	searchFilter      string
	searchInput       textinput.Model
	logCollector      *resources.LogCollector
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
	if len(m.searchFilter) > 0 {
		label += " " + m.searchFilter
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

func NewProgram(watcher *resources.K8sWatcher, d dao.KhronoStore, l *resources.LogCollector) *KhronoscopeTeaProgram {
	am := &KhronoscopeTeaProgram{
		watcher:      watcher,
		data:         d,
		tv:           ui.NewTreeView(),
		logCollector: l,
	}

	return am
}

func (s *KhronoscopeTeaProgram) Init() tea.Cmd { return nil }

func (m *KhronoscopeTeaProgram) View() string {
	timeToUse := m.VCR.GetTimeToUse()
	resources := m.data.GetResourcesAt(timeToUse, "", "")
	convResources := make([]types.Resource, len(resources))
	for i := 0; i < len(resources); i++ {
		convResources[i] = resources[i]
	}
	m.tv.UpdateResources(convResources)

	currentLabel := m.data.GetLabel(timeToUse)

	m.tv.SetFilter(m.searchFilter)
	treeContent, focusLine := m.tv.Render(m.logCollector)
	treeContent = lipgloss.NewStyle().Width(m.treeView.Width).Render(treeContent)
	m.treeView.SetContent(treeContent)
	m.treeView.YOffset = focusLine - (m.treeView.Height / 2)
	if m.treeView.YOffset < 0 {
		m.treeView.YOffset = 0
	}

	resource := m.tv.GetSelected()
	if resource != nil {
		detailContent := fmt.Sprintf("UID: %s\n", resource.GetUID()) + strings.Join(resource.GetDetails(), "\n")
		detailContent = lipgloss.NewStyle().Width(m.detailView.Width).Render(detailContent)
		m.detailView.SetContent(detailContent)
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
				m.searchFilter = m.searchInput.Value()
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
		case "l":
			m.logCollector.ToggleLogs(m.tv.GetSelected())
		case "/":
			m.searchInput = textinput.New()
			m.searchInput.Placeholder = ""
			m.searchInput.Focus()
			m.searchInput.CharLimit = 156
			m.searchInput.Width = 20
			m.search = true
		case "s":
			m.data.Save("temp.dat")
		case "1":
			m.SetPopup(ui.NewMessagePopup("Hello World!\nThis is a popup\nYay!", "esc"))
			return m, nil
		case "m":
			m.VCR.Pause()
			m.SetPopup(ui.NewLabelPopup(m))
		case "tab":
			m.viewMode++
			m.viewMode %= 2
			m.windowResize(m.lastWindowSizeMsg)
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		case "left":
			m.VCR.Rewind()
			return m, nil
		case "right":
			m.VCR.FastForward()
			return m, nil
		case " ":
			if m.VCR.GetPlaySpeed() == 0 {
				m.VCR.Play()
			} else {
				m.VCR.Pause()
			}
			return m, nil
		case "esc":
			m.VCR.DisableVirtualTime()
		case "enter":
			m.tv.Toggle()
			return m, nil
		case "shift+up":
			m.detailView.LineUp(10)
			return m, nil
		case "shift+down":
			m.detailView.LineDown(10)
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
