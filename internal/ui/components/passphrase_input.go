package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PassphraseConfirmedMsg is sent when a passphrase is confirmed
type PassphraseConfirmedMsg struct {
	Passphrase string
}

// PassphraseCancelledMsg is sent when passphrase input is cancelled
type PassphraseCancelledMsg struct{}

// PassphraseInput is a component for inputting and confirming passphrases
type PassphraseInput struct {
	textInput        textinput.Model
	confirmInput     textinput.Model
	title            string
	showConfirmation bool
	width            int
	errMsg           string
}

// NewPassphraseInput creates a new passphrase input component
func NewPassphraseInput(title string, requireConfirmation bool) *PassphraseInput {
	ti := textinput.New()
	ti.Placeholder = "Enter passphrase"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()

	confirm := textinput.New()
	confirm.Placeholder = "Confirm passphrase"
	confirm.EchoMode = textinput.EchoPassword
	confirm.EchoCharacter = '•'

	return &PassphraseInput{
		textInput:        ti,
		confirmInput:     confirm,
		title:            title,
		showConfirmation: requireConfirmation,
		width:            40,
	}
}

// Init initializes the component
func (p PassphraseInput) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles events and updates the model
func (p *PassphraseInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return p, func() tea.Msg { return PassphraseCancelledMsg{} }

		case tea.KeyEnter:
			// If confirmation is not shown yet but required, show it
			if p.showConfirmation && !p.confirmInput.Focused() {
				p.textInput.Blur()
				p.confirmInput.Focus()
				return p, textinput.Blink
			}

			// If confirmation is shown, check that passphrases match
			if p.confirmInput.Focused() {
				if p.textInput.Value() != p.confirmInput.Value() {
					p.errMsg = "Passphrases do not match"
					p.confirmInput.Reset()
					return p, textinput.Blink
				}
			}

			// Passphrase is confirmed (either no confirmation needed or passphrases match)
			return p, func() tea.Msg {
				return PassphraseConfirmedMsg{Passphrase: p.textInput.Value()}
			}

		case tea.KeyTab:
			if p.showConfirmation {
				if p.textInput.Focused() {
					p.textInput.Blur()
					p.confirmInput.Focus()
				} else {
					p.confirmInput.Blur()
					p.textInput.Focus()
				}
				return p, textinput.Blink
			}
		}
	}

	// Update the active text input
	if p.textInput.Focused() {
		p.textInput, cmd = p.textInput.Update(msg)
		cmds = append(cmds, cmd)
	} else if p.confirmInput.Focused() {
		p.confirmInput, cmd = p.confirmInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return p, tea.Batch(cmds...)
}

// View renders the component
func (p PassphraseInput) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).MarginBottom(1)
	inputStyle := lipgloss.NewStyle().Width(p.width).PaddingLeft(1)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).PaddingTop(1)

	view := titleStyle.Render(p.title) + "\n"
	view += inputStyle.Render(p.textInput.View()) + "\n"

	if p.showConfirmation {
		view += inputStyle.Render(p.confirmInput.View()) + "\n"
	}

	if p.errMsg != "" {
		view += errorStyle.Render(p.errMsg) + "\n"
	}

	view += "\nPress Enter to confirm or Esc to cancel"

	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1).Render(view)
}

// SetWidth sets the width of the input field
func (p *PassphraseInput) SetWidth(width int) {
	p.width = width
}

