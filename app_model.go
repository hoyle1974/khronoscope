package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hoyle1974/khronoscope/internal/ui"
	"github.com/hoyle1974/khronoscope/resources"
)

// Program holds the reference to the Bubbletea program
var Program *tea.Program

type AppModel struct {
	data              DataModel
	watcher           *resources.K8sWatcher
	ready             bool
	viewMode          int
	width             int
	height            int
	lastWindowSizeMsg tea.WindowSizeMsg
	vcr               *VCRControl
	popup             Popup

	// UI Components
	headerView *ui.HeaderView
	footerView *ui.FooterView
	treeView   *ui.TreeView
	detailView *ui.DetailView
	popupView  *ui.PopupView
}

func (m *AppModel) SetLabel(label string) {
	m.data.SetLabel(m.vcr.GetTimeToUse(), label)
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

func (m *AppModel) SetPopup(popup Popup) {
	m.popup = popup
}

func newModel(watcher *resources.K8sWatcher, data DataModel) *AppModel {
	m := &AppModel{
		data:       data,
		watcher:    watcher,
		headerView: ui.NewHeaderView(),
		footerView: ui.NewFooterView(0),
		treeView:   ui.NewTreeView(0, 0),
		detailView: ui.NewDetailView(0, 0),
		popupView:  ui.NewPopupView(0, 0),
	}
	m.vcr = NewVCRControl(data, func() {
		if Program != nil {
			Program.Send(1)
		}
	})
	return m
}

func (m *AppModel) View() string {
	timeToUse := m.vcr.GetTimeToUse()

	// Update UI components
	minTime, maxTime := m.data.GetTimeRange()
	m.headerView.Update(minTime, maxTime, timeToUse, m.vcr.IsEnabled())
	m.footerView.Update(m.width, m.treeView.ScrollPercent())

	resourceList := m.data.GetResourcesAt(timeToUse, "", "")
	m.treeView.Update(m.width, m.height, resourceList, timeToUse)

	var selectedResource *resources.Resource
	if len(resourceList) > 0 {
		selectedResource = &resourceList[0]
	}
	m.detailView.Update(m.width, m.height, selectedResource)

	// Render the layout
	var content string
	if m.viewMode == 0 {
		content = lipgloss.JoinHorizontal(lipgloss.Left,
			m.treeView.View(),
			m.detailView.View(),
		)
	} else {
		content = lipgloss.JoinVertical(lipgloss.Top,
			m.treeView.View(),
			m.detailView.View(),
		)
	}

	fullView := fmt.Sprintf("%s\n%s\n%s",
		m.headerView.Render(),
		content,
		m.footerView.Render(),
	)

	if m.popup != nil {
		return m.popupView.InsertPopup(fullView, m.popup)
	}
	return fullView
}

func (m *AppModel) windowResize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.lastWindowSizeMsg = msg

	headerHeight := 1
	footerHeight := 1
	vcrHeight := 3

	contentHeight := m.height - headerHeight - footerHeight - vcrHeight
	contentWidth := m.width

	if m.viewMode == 0 {
		// Split view
		m.treeView.Update(contentWidth/2, contentHeight, nil, time.Time{})
		m.detailView.Update(contentWidth/2-1, contentHeight, nil)
	} else {
		// Full view
		m.treeView.Update(contentWidth, contentHeight/2, nil, time.Time{})
		m.detailView.Update(contentWidth-1, contentHeight/2, nil)
	}

	m.popupView.Update(m.width, m.height)
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.popup != nil {
		if m.popup.Update(msg) {
			m.SetPopup(nil)
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			m.data.Save("temp.dat")
		case "1":
			m.SetPopup(newMessagePopup("Hello World!\nThis is a popup\nYay!", "esc"))
			return m, nil
		case "l":
			m.vcr.Pause()
			m.SetPopup(NewLabelPopup(m))
		case "tab":
			m.viewMode++
			m.viewMode %= 2
			m.windowResize(m.lastWindowSizeMsg)
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		case "left":
			m.vcr.Rewind()
			return m, nil
		case "right":
			m.vcr.FastForward()
			return m, nil
		case " ":
			if m.vcr.playSpeed == 0 {
				m.vcr.Play()
			} else {
				m.vcr.Pause()
			}
			return m, nil
		case "esc":
			m.vcr.DisableVCR()
		case "enter":
			m.treeView.Toggle()
			return m, nil
		case "shift+up":
			m.detailView.LineUp(10)
			return m, nil
		case "shift+down":
			m.detailView.LineDown(10)
			return m, nil
		case "up":
			m.treeView.Up()
			return m, nil
		case "down":
			m.treeView.Down()
			return m, nil
		case "alt+up":
			m.treeView.PageUp()
			return m, nil
		case "alt+down":
			m.treeView.PageDown()
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.windowResize(msg)
	case int:
		timeToUse := m.vcr.GetTimeToUse()
		resourceList := m.data.GetResourcesAt(timeToUse, "", "")
		m.treeView.Update(m.width, m.height, resourceList, timeToUse)
		return m, tea.Batch(cmds...)
	}

	// Handle keyboard and mouse events in the viewport
	// s.viewport, cmd = s.viewport.Update(msg)
	// cmds = append(cmds, cmd)

	// return s, tea.Batch(cmds...)
	return m, nil
}

// Init implements tea.Model
func (m *AppModel) Init() tea.Cmd {
	// Return nil if no initial commands need to be run
	return nil
}
