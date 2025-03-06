package views

import (
	"fmt"
	"os"
	"time"

	"github.com/bxtal-lsn/supper/internal/age"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DashboardView is the main dashboard view
type DashboardView struct {
	keys            KeyMap
	viewport        viewport.Model
	width           int
	height          int
	hasDecryptedKey bool
	hasEncryptedKey bool
	keyPath         string
	encryptedPath   string
	keyCreated      time.Time
	keyExpiry       time.Time
	publicKey       string
}

// NewDashboardView creates a new dashboard view
func NewDashboardView() *DashboardView {
	return &DashboardView{
		keys:          DefaultKeyMap(),
		keyPath:       age.DefaultKeyPath(),
		encryptedPath: age.DefaultEncryptedKeyPath(),
		keyExpiry:     time.Now().Add(12 * time.Hour), // Placeholder
	}
}

// Init initializes the view
func (d *DashboardView) Init() tea.Cmd {
	return d.checkKeyStatus()
}

// Update handles events and updates the model
func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		d.viewport = viewport.New(msg.Width, msg.Height-5)
		d.viewport.YPosition = 2

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keys.GenerateKey):
			return d, func() tea.Msg {
				return SwitchTabMsg{Tab: ViewKeyManager}
			}

		case key.Matches(msg, d.keys.DecryptKey) && d.hasEncryptedKey:
			return d, func() tea.Msg {
				return SwitchTabMsg{Tab: ViewKeyManager}
			}
		}
	}

	d.viewport, cmd = d.viewport.Update(msg)
	cmds = append(cmds, cmd)

	// Periodically check key status
	cmds = append(cmds, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return CheckKeyStatusMsg{}
	}))

	return d, tea.Batch(cmds...)
}

// View renders the view
func (d *DashboardView) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#1E88E5")).Padding(0, 1)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#1E88E5")).
		Padding(1, 2).
		Width(60)

	// Key status section
	keyStatus := "Key Status: "
	if d.hasDecryptedKey {
		keyStatus += lipgloss.NewStyle().Foreground(lipgloss.Color("#00AA00")).Render("Decrypted")
	} else if d.hasEncryptedKey {
		keyStatus += lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00")).Render("Encrypted")
	} else {
		keyStatus += lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("Not found")
	}

	// Calculate time remaining if key is decrypted
	var timeRemaining string
	if d.hasDecryptedKey {
		remaining := d.keyExpiry.Sub(time.Now())
		if remaining > 0 {
			timeRemaining = fmt.Sprintf("Auto-delete in: %s", remaining.Round(time.Second))
		} else {
			timeRemaining = "Key will be deleted soon"
		}
	}

	keySection := boxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render("Age Key"),
			"",
			keyStatus,
			timeRemaining,
			"",
			fmt.Sprintf("Key path: %s", d.keyPath),
			fmt.Sprintf("Encrypted path: %s", d.encryptedPath),
			"",
			d.getKeyActions(),
		),
	)

	// Recent files section
	recentFilesSection := boxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render("Recent Files"),
			"",
			"No recent files",
			"",
			"Press 'f' to browse files",
		),
	)

	// Quick Actions
	quickActionsSection := boxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render("Quick Actions"),
			"",
			"g - Generate new key",
			"d - Decrypt key",
			"e - Encrypt file",
			"D - Decrypt file",
			"E - Edit file",
		),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Dashboard"),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.JoinVertical(
				lipgloss.Left,
				keySection,
				quickActionsSection,
			),
			recentFilesSection,
		),
	)
}

// getKeyActions returns actions based on key status
func (d *DashboardView) getKeyActions() string {
	if d.hasDecryptedKey {
		return "Press 'x' to securely delete the decrypted key"
	} else if d.hasEncryptedKey {
		return "Press 'd' to decrypt your key"
	}
	return "Press 'g' to generate a new key"
}

// checkKeyStatus checks if keys exist
func (d *DashboardView) checkKeyStatus() tea.Cmd {
	return func() tea.Msg {
		// Check if decrypted key exists
		_, err := os.Stat(d.keyPath)
		d.hasDecryptedKey = err == nil

		// Check if encrypted key exists
		_, err = os.Stat(d.encryptedPath)
		d.hasEncryptedKey = err == nil

		// If decrypted key exists, get info about it
		if d.hasDecryptedKey {
			fileInfo, err := os.Stat(d.keyPath)
			if err == nil {
				d.keyCreated = fileInfo.ModTime()
				// This is a placeholder - in a real app, you'd parse the key to get more info
				d.publicKey = "age1..."
			}
		}

		return nil
	}
}

// SwitchTabMsg is sent to switch tabs
type SwitchTabMsg struct {
	Tab int
}

// CheckKeyStatusMsg is sent to check key status
type CheckKeyStatusMsg struct{}
