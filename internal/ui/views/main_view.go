package views

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View types
const (
	ViewDashboard = iota
	ViewKeyManager
	ViewFileBrowser
	ViewSettings
)

// KeyMap defines the keybindings for the application
type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	Help        key.Binding
	Quit        key.Binding
	Tab         key.Binding
	ShiftTab    key.Binding
	Enter       key.Binding
	GenerateKey key.Binding
	DecryptKey  key.Binding
	EncryptFile key.Binding
	DecryptFile key.Binding
	EditFile    key.Binding
	DeleteKey   key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "move left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "move right"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous tab"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		GenerateKey: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "generate key"),
		),
		DecryptKey: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "decrypt key"),
		),
		EncryptFile: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "encrypt file"),
		),
		DecryptFile: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "decrypt file"),
		),
		EditFile: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "edit file"),
		),
		DeleteKey: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete key"),
		),
	}
}

// MainView represents the main view of the application
type MainView struct {
	keys           KeyMap
	help           help.Model
	viewport       viewport.Model
	currentTab     int
	width          int
	height         int
	ready          bool
	dashboardView  *DashboardView
	keyManagerView *KeyManagerView
	fileEditorView *FileEditorView
	settingsView   *SettingsView
}

// NewMainView creates a new main view
func NewMainView() *MainView {
	keys := DefaultKeyMap()
	h := help.New()

	// Initialize sub-views
	dashboardView := NewDashboardView()
	keyManagerView := NewKeyManagerView()
	fileEditorView := NewFileEditorView()
	settingsView := NewSettingsView()

	return &MainView{
		keys:           keys,
		help:           h,
		currentTab:     ViewDashboard,
		dashboardView:  dashboardView,
		keyManagerView: keyManagerView,
		fileEditorView: fileEditorView,
		settingsView:   settingsView,
	}
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (m MainView) ShortHelp() []key.Binding {
	kb := []key.Binding{
		m.keys.Help,
		m.keys.Quit,
		m.keys.Tab,
	}

	// Add view-specific keybindings based on current tab
	switch m.currentTab {
	case ViewDashboard:
		kb = append(kb, m.keys.GenerateKey, m.keys.DecryptKey)
	case ViewKeyManager:
		kb = append(kb, m.keys.GenerateKey, m.keys.DecryptKey, m.keys.DeleteKey)
	case ViewFileBrowser:
		kb = append(kb, m.keys.EncryptFile, m.keys.DecryptFile, m.keys.EditFile)
	}

	return kb
}

// FullHelp returns keybindings for the expanded help view
func (m MainView) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.Up, m.keys.Down, m.keys.Left, m.keys.Right},
		{m.keys.Tab, m.keys.ShiftTab, m.keys.Enter},
		{m.keys.GenerateKey, m.keys.DecryptKey, m.keys.DeleteKey},
		{m.keys.EncryptFile, m.keys.DecryptFile, m.keys.EditFile},
		{m.keys.Help, m.keys.Quit},
	}
}

// tabStyle returns the style for tab headings
func (m MainView) tabStyle(selected bool) lipgloss.Style {
	base := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true)

	if selected {
		return base.
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#1E88E5"))
	}

	return base.
		Foreground(lipgloss.Color("#AAAAAA"))
}

// Init initializes the main view
func (m MainView) Init() tea.Cmd {
	return tea.Batch(
		m.dashboardView.Init(),
		m.keyManagerView.Init(),
		m.fileEditorView.Init(),
		m.settingsView.Init(),
	)
}

// Update handles events and updates the model
func (m *MainView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 3
		footerHeight := 3
		m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
		m.viewport.YPosition = headerHeight
		m.ready = true

		// Propagate window size to sub-views
		var subMsg tea.Msg = msg

		// Update dashboard view
		dashModel, dashCmd := m.dashboardView.Update(subMsg)
		if updatedModel, ok := dashModel.(*DashboardView); ok {
			m.dashboardView = updatedModel
		}
		cmds = append(cmds, dashCmd)

		// Update key manager view
		keyModel, keyCmd := m.keyManagerView.Update(subMsg)
		if updatedModel, ok := keyModel.(*KeyManagerView); ok {
			m.keyManagerView = updatedModel
		}
		cmds = append(cmds, keyCmd)

		// Update file editor view
		fileModel, fileCmd := m.fileEditorView.Update(subMsg)
		if updatedModel, ok := fileModel.(*FileEditorView); ok {
			m.fileEditorView = updatedModel
		}
		cmds = append(cmds, fileCmd)

		// Update settings view
		settingsModel, settingsCmd := m.settingsView.Update(subMsg)
		if updatedModel, ok := settingsModel.(*SettingsView); ok {
			m.settingsView = updatedModel
		}
		cmds = append(cmds, settingsCmd)

	case tea.KeyMsg:
		// Global key handlers
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Tab):
			m.currentTab = (m.currentTab + 1) % 4 // Cycle through tabs

		case key.Matches(msg, m.keys.ShiftTab):
			m.currentTab = (m.currentTab - 1 + 4) % 4 // Cycle backwards

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		}

	case SwitchTabMsg:
		// Handle tab switching from sub-views
		m.currentTab = msg.Tab
		return m, nil

	case CheckKeyStatusMsg:
		// Propagate key status check to all views
		dashModel, dashCmd := m.dashboardView.Update(msg)
		if updatedModel, ok := dashModel.(*DashboardView); ok {
			m.dashboardView = updatedModel
		}
		cmds = append(cmds, dashCmd)

		keyModel, keyCmd := m.keyManagerView.Update(msg)
		if updatedModel, ok := keyModel.(*KeyManagerView); ok {
			m.keyManagerView = updatedModel
		}
		cmds = append(cmds, keyCmd)

		fileModel, fileCmd := m.fileEditorView.Update(msg)
		if updatedModel, ok := fileModel.(*FileEditorView); ok {
			m.fileEditorView = updatedModel
		}
		cmds = append(cmds, fileCmd)
	}

	// Update the active sub-view
	switch m.currentTab {
	case ViewDashboard:
		var dashModel tea.Model
		dashModel, cmd = m.dashboardView.Update(msg)
		if updatedModel, ok := dashModel.(*DashboardView); ok {
			m.dashboardView = updatedModel
		}
		cmds = append(cmds, cmd)

	case ViewKeyManager:
		var keyModel tea.Model
		keyModel, cmd = m.keyManagerView.Update(msg)
		if updatedModel, ok := keyModel.(*KeyManagerView); ok {
			m.keyManagerView = updatedModel
		}
		cmds = append(cmds, cmd)

	case ViewFileBrowser:
		var fileModel tea.Model
		fileModel, cmd = m.fileEditorView.Update(msg)
		if updatedModel, ok := fileModel.(*FileEditorView); ok {
			m.fileEditorView = updatedModel
		}
		cmds = append(cmds, cmd)

	case ViewSettings:
		var settingsModel tea.Model
		settingsModel, cmd = m.settingsView.Update(msg)
		if updatedModel, ok := settingsModel.(*SettingsView); ok {
			m.settingsView = updatedModel
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the application UI
func (m MainView) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Create tab bar
	tabs := []string{"Dashboard", "Key Manager", "Files", "Settings"}
	tabsView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.tabStyle(m.currentTab == ViewDashboard).Render(tabs[0]),
		m.tabStyle(m.currentTab == ViewKeyManager).Render(tabs[1]),
		m.tabStyle(m.currentTab == ViewFileBrowser).Render(tabs[2]),
		m.tabStyle(m.currentTab == ViewSettings).Render(tabs[3]),
	)

	// Render content based on current tab
	var content string
	switch m.currentTab {
	case ViewDashboard:
		content = m.dashboardView.View()
	case ViewKeyManager:
		content = m.keyManagerView.View()
	case ViewFileBrowser:
		content = m.fileEditorView.View()
	case ViewSettings:
		content = m.settingsView.View()
	}

	// Combine all elements
	var helpView string
	if m.help.ShowAll {
		helpView = m.help.View(m)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		tabsView,
		content,
		helpView,
	)
}

