package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AppModel struct {
	watcher           *K8sWatcher
	ready             bool
	viewMode          int
	width             int
	height            int
	treeView          viewport.Model
	detailView        viewport.Model
	lastWindowSizeMsg tea.WindowSizeMsg
	tv                *TreeView
}

func (m *AppModel) headerView() string {
	title := titleStyle.Render(fmt.Sprintf("Khronoscope - %s", GetTimeToUse().Format("2006-01-02 15:04:05")))
	line := strings.Repeat("─", max(0, m.width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m *AppModel) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.treeView.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

// MODEL DATA
var useAdjustment = false
var adjust = time.Now()

func newModel(watcher *K8sWatcher) *AppModel {
	return &AppModel{
		watcher: watcher,
		tv:      NewTreeView(),
	}
}

func (s *AppModel) Init() tea.Cmd { return nil }

// VIEW

var curPosition = 0
var curRealPosition = 0
var count = 0

func GetTimeToUse() time.Time {
	if useAdjustment {
		return adjust
	}
	return time.Now()
}

func (m *AppModel) View() string {
	timeToUse := GetTimeToUse()
	m.tv.AddResources(m.watcher.GetStateAtTime(timeToUse, "", ""))

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
		temp = lipgloss.JoinVertical(0, fixWidth(m.treeView.View(), m.width), " ", m.detailView.View())
	}

	// log := fmt.Sprintf("%d : %v - %v\n", count, adjust.Seconds(), watcher.GetLog())

	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), temp, m.footerView())
}

// UPDATE
func (m *AppModel) windowResize(msg tea.WindowSizeMsg) {
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

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if !useAdjustment {
				useAdjustment = true
				adjust = time.Now()
			} else {
				adjust = adjust.Add(-time.Second)
			}
			return m, nil
		case "right":
			if !useAdjustment {
				useAdjustment = true
				adjust = time.Now()
			} else {
				adjust = adjust.Add(time.Second)
				if adjust.After(time.Now()) {
					useAdjustment = false
				}
			}
			return m, nil
		case "esc":
			useAdjustment = false
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