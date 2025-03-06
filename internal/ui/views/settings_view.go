package views

import (
	"fmt"
	"time"

	"github.com/bxtal-lsn/supper/internal/age"
	"github.com/bxtal-lsn/supper/internal/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SettingItem represents a setting in the settings view
type SettingItem struct {
	Name        string
	Description string
	Value       string
	Editable    bool
	InputField  textinput.Model
}

// SettingsView is the view for application settings
type SettingsView struct {
	keys       KeyMap
	viewport   viewport.Model
	width      int
	height     int
	settings   []SettingItem
	cursor     int
	editingIdx int
	err        error
}

// NewSettingsView creates a new settings view
func NewSettingsView() *SettingsView {
	// Create settings
	settings := []SettingItem{
		{
			Name:        "Age Key Path",
			Description: "Path to the age key file",
			Value:       age.DefaultKeyPath(),
			Editable:    true,
		},
		{
			Name:        "Encrypted Key Path",
			Description: "Path to the encrypted age key file",
			Value:       age.DefaultEncryptedKeyPath(),
			Editable:    true,
		},
		{
			Name:        "Auto-Delete Interval",
			Description: "Automatically delete decrypted key after this time",
			Value:       "30m",
			Editable:    true,
		},
		{
			Name:        "Editor Command",
			Description: "Command to use for editing files",
			Value:       "default",
			Editable:    true,
		},
		{
			Name:        "Default Recipients",
			Description: "Default age recipients for new files",
			Value:       "",
			Editable:    true,
		},
	}

	// Initialize input fields
	for i := range settings {
		if settings[i].Editable {
			input := textinput.New()
			input.Placeholder = settings[i].Value
			input.Width = 40
			settings[i].InputField = input
		}
	}

	return &SettingsView{
		keys:       DefaultKeyMap(),
		settings:   settings,
		cursor:     0,
		editingIdx: -1,
	}
}

// Init initializes the view
func (s *SettingsView) Init() tea.Cmd {
	return s.loadSettings()
}

// Update handles events and updates the model
func (s *SettingsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.viewport = viewport.New(msg.Width, msg.Height-5)
		s.viewport.YPosition = 2

	case tea.KeyMsg:
		// If currently editing a setting
		if s.editingIdx >= 0 {
			switch msg.Type {
			case tea.KeyEnter:
				// Save the edited value
				s.settings[s.editingIdx].Value = s.settings[s.editingIdx].InputField.Value()
				s.editingIdx = -1
				// Save all settings
				return s, s.saveSettings()

			case tea.KeyEsc:
				// Cancel editing
				s.editingIdx = -1
				return s, nil
			}

			// Update the input field
			s.settings[s.editingIdx].InputField, cmd = s.settings[s.editingIdx].InputField.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			// Regular navigation
			switch {
			case key.Matches(msg, s.keys.Up):
				s.cursor = max(0, s.cursor-1)

			case key.Matches(msg, s.keys.Down):
				s.cursor = min(len(s.settings)-1, s.cursor+1)

			case key.Matches(msg, s.keys.Enter):
				if s.settings[s.cursor].Editable {
					s.editingIdx = s.cursor
					s.settings[s.editingIdx].InputField.SetValue(s.settings[s.editingIdx].Value)
					s.settings[s.editingIdx].InputField.Focus()
					return s, textinput.Blink
				}
			}
		}
	}

	// Update viewport
	if s.viewport.Height > 0 {
		s.viewport, cmd = s.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return s, tea.Batch(cmds...)
}

// View renders the view
func (s *SettingsView) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#1E88E5")).Padding(0, 1)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#1E88E5"))
	normalStyle := lipgloss.NewStyle()
	descriptionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00AA00"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

	content := ""

	// Render each setting
	for i, setting := range s.settings {
		var row string
		isSelected := i == s.cursor
		isEditing := i == s.editingIdx

		// Format the row
		if isEditing {
			// If we're editing this setting, show the input field
			row = fmt.Sprintf("%s: %s",
				setting.Name,
				setting.InputField.View(),
			)
			row = normalStyle.Render(row)
		} else {
			// Regular display
			var style lipgloss.Style
			if isSelected {
				style = selectedStyle
			} else {
				style = normalStyle
			}

			row = fmt.Sprintf("%s: %s",
				style.Render(setting.Name),
				valueStyle.Render(setting.Value),
			)
		}

		// Add description
		if isSelected || isEditing {
			row += "\n" + descriptionStyle.Render("  "+setting.Description)
		}

		content += row + "\n\n"
	}

	// Add help text
	helpText := "↑/↓: Navigate • Enter: Edit • Esc: Cancel"
	if s.editingIdx >= 0 {
		helpText = "Enter: Save • Esc: Cancel"
	}

	// Add error message if present
	if s.err != nil {
		content += errorStyle.Render(fmt.Sprintf("Error: %v", s.err)) + "\n\n"
	}

	// Combine all elements
	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Settings"),
		content,
		descriptionStyle.Render(helpText),
	)
}

// loadSettings loads settings from the configuration
func (s *SettingsView) loadSettings() tea.Cmd {
	return func() tea.Msg {
		// This would be replaced with actual configuration loading logic
		cfg, err := config.Load()
		if err != nil {
			s.err = fmt.Errorf("failed to load settings: %w", err)
			return nil
		}

		// Update settings with loaded values
		for i, setting := range s.settings {
			switch setting.Name {
			case "Age Key Path":
				s.settings[i].Value = cfg.KeyPath
			case "Encrypted Key Path":
				s.settings[i].Value = cfg.EncryptedKeyPath
			case "Auto-Delete Interval":
				s.settings[i].Value = cfg.AutoDeleteInterval.String()
			case "Editor Command":
				s.settings[i].Value = cfg.EditorCommand
			case "Default Recipients":
				s.settings[i].Value = cfg.DefaultRecipients
			}
		}

		return nil
	}
}

// saveSettings saves the current settings
func (s *SettingsView) saveSettings() tea.Cmd {
	return func() tea.Msg {
		// Create a configuration object
		cfg := &config.Config{}

		// Update with current values
		for _, setting := range s.settings {
			switch setting.Name {
			case "Age Key Path":
				cfg.KeyPath = setting.Value
			case "Encrypted Key Path":
				cfg.EncryptedKeyPath = setting.Value
			case "Auto-Delete Interval":
				duration, err := time.ParseDuration(setting.Value)
				if err != nil {
					s.err = fmt.Errorf("invalid duration format for Auto-Delete Interval: %w", err)
					return nil
				}
				cfg.AutoDeleteInterval = duration
			case "Editor Command":
				cfg.EditorCommand = setting.Value
			case "Default Recipients":
				cfg.DefaultRecipients = setting.Value
			}
		}

		// Save the configuration
		if err := config.Save(cfg); err != nil {
			s.err = fmt.Errorf("failed to save settings: %w", err)
			return nil
		}

		return nil
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

