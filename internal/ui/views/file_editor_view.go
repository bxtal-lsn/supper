package views

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bxtal-lsn/supper/internal/age"
	"github.com/bxtal-lsn/supper/internal/sops"
	"github.com/bxtal-lsn/supper/internal/ui/components"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileEditor states
const (
	stateFileSelect int = iota
	stateRecipientInput
	stateEncrypting
	stateDecrypting
	stateEditing
	stateConfirmation
	stateComplete
	stateError
)

// FileEditorView is the view for encrypting, decrypting, and editing files
type FileEditorView struct {
	keys            KeyMap
	viewport        viewport.Model
	spinner         spinner.Model
	fileBrowser     *components.FileBrowser
	textInput       textinput.Model
	width           int
	height          int
	state           int
	selectedFile    string
	fileInfo        *sops.FileInfo
	recipientInput  string
	operation       string
	operationResult string
	error           error
	showHelp        bool
	hasDecryptedKey bool
}

// NewFileEditorView creates a new file editor view
func NewFileEditorView() *FileEditorView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Enter age recipient public key"
	ti.Width = 50

	fb := components.NewFileBrowser()

	return &FileEditorView{
		keys:        DefaultKeyMap(),
		spinner:     s,
		fileBrowser: fb,
		textInput:   ti,
		state:       stateFileSelect,
		showHelp:    true,
	}
}

// Init initializes the view
func (f *FileEditorView) Init() tea.Cmd {
	return tea.Batch(
		f.fileBrowser.Init(),
		f.spinner.Tick,
		f.checkKeyStatus(),
	)
}

// Update handles events and updates the model
func (f *FileEditorView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height
		f.viewport = viewport.New(msg.Width, msg.Height-5)
		f.viewport.YPosition = 2
		f.fileBrowser.SetSize(msg.Width, msg.Height-10)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, f.keys.Quit):
			if f.state == stateFileSelect {
				return f, tea.Quit
			} else {
				// Go back to file select state
				f.state = stateFileSelect
				f.error = nil
				return f, nil
			}

		case key.Matches(msg, f.keys.Help):
			f.showHelp = !f.showHelp

		case key.Matches(msg, f.keys.EncryptFile) && f.state == stateFileSelect:
			if f.selectedFile != "" && (!f.fileInfo.Encrypted) {
				f.state = stateRecipientInput
				f.operation = "encrypt"
				f.textInput.Focus()
				return f, nil
			}

		case key.Matches(msg, f.keys.DecryptFile) && f.state == stateFileSelect:
			if f.selectedFile != "" && f.fileInfo.Encrypted && f.hasDecryptedKey {
				f.state = stateConfirmation
				f.operation = "decrypt"
				return f, nil
			}

		case key.Matches(msg, f.keys.EditFile) && f.state == stateFileSelect:
			if f.selectedFile != "" && f.fileInfo.Encrypted && f.hasDecryptedKey {
				f.state = stateConfirmation
				f.operation = "edit"
				return f, nil
			}

		case key.Matches(msg, f.keys.Enter):
			switch f.state {
			case stateRecipientInput:
				if f.textInput.Value() != "" {
					f.recipientInput = f.textInput.Value()
					f.state = stateConfirmation
				}
			case stateConfirmation:
				switch f.operation {
				case "encrypt":
					f.state = stateEncrypting
					return f, f.encryptFile()
				case "decrypt":
					f.state = stateDecrypting
					return f, f.decryptFile()
				case "edit":
					f.state = stateEditing
					return f, f.editFile()
				}
			case stateComplete, stateError:
				f.state = stateFileSelect
				f.error = nil
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		f.spinner, cmd = f.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case components.FileSelectedMsg:
		f.selectedFile = msg.Path
		f.fileInfo = msg.Info
		if f.fileInfo == nil {
			// If no file info (shouldn't happen), create a default one
			f.fileInfo = &sops.FileInfo{
				Path:      msg.Path,
				Encrypted: false,
			}
		}

	case OperationCompleteMsg:
		f.state = stateComplete
		f.operationResult = msg.Message

	case OperationErrorMsg:
		f.state = stateError
		f.error = msg.Error
	}

	// Update sub-components based on state
	switch f.state {
	case stateFileSelect:
		newModel, cmd := f.fileBrowser.Update(msg)
		if updatedModel, ok := newModel.(*components.FileBrowser); ok {
			f.fileBrowser = updatedModel
		}
		cmds = append(cmds, cmd)

	case stateRecipientInput:
		f.textInput, cmd = f.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return f, tea.Batch(cmds...)
}

// View renders the view
func (f *FileEditorView) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#1E88E5")).Padding(0, 1)

	var content string
	switch f.state {
	case stateFileSelect:
		content = f.fileBrowser.View()

		// Show file info if a file is selected
		if f.selectedFile != "" {
			infoStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1)

			fileInfo := fmt.Sprintf("Selected: %s\n", f.selectedFile)
			fileInfo += fmt.Sprintf("Status: %s\n", getEncryptionStatusText(f.fileInfo))

			// Show available actions based on file state
			fileInfo += "\nAvailable Actions:\n"
			if !f.fileInfo.Encrypted {
				fileInfo += "  e - Encrypt file\n"
			}
			if f.fileInfo.Encrypted && f.hasDecryptedKey {
				fileInfo += "  d - Decrypt file\n"
				fileInfo += "  E - Edit file\n"
			}

			content = lipgloss.JoinVertical(
				lipgloss.Left,
				content,
				infoStyle.Render(fileInfo),
			)
		}

	case stateRecipientInput:
		content = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				"Enter the age public key of the recipient:",
				f.textInput.View(),
				"",
				"Press Enter to confirm or Esc to cancel",
			),
		)

	case stateConfirmation:
		confirmStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1)

		var action string
		switch f.operation {
		case "encrypt":
			action = fmt.Sprintf("encrypt file %s for recipient %s", f.selectedFile, f.recipientInput)
		case "decrypt":
			action = fmt.Sprintf("decrypt file %s", f.selectedFile)
		case "edit":
			action = fmt.Sprintf("edit encrypted file %s", f.selectedFile)
		}

		content = confirmStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				fmt.Sprintf("Are you sure you want to %s?", action),
				"",
				"Press Enter to confirm or Esc to cancel",
			),
		)

	case stateEncrypting, stateDecrypting, stateEditing:
		var operation string
		switch f.state {
		case stateEncrypting:
			operation = "Encrypting"
		case stateDecrypting:
			operation = "Decrypting"
		case stateEditing:
			operation = "Opening"
		}

		content = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				fmt.Sprintf("%s %s file...", f.spinner.View(), operation),
				fmt.Sprintf("File: %s", f.selectedFile),
			),
		)

	case stateComplete:
		content = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00AA00")).
			Padding(1).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					"Operation complete!",
					"",
					f.operationResult,
					"",
					"Press Enter to continue",
				),
			)

	case stateError:
		content = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF0000")).
			Padding(1).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					"Error:",
					"",
					fmt.Sprintf("%v", f.error),
					"",
					"Press Enter to continue",
				),
			)
	}

	// Show help if enabled
	var helpView string
	if f.showHelp {
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))
		helpContent := "? - toggle help, q - quit/back"

		switch f.state {
		case stateFileSelect:
			helpContent += ", e - encrypt, d - decrypt, E - edit"
		case stateRecipientInput, stateConfirmation:
			helpContent += ", Enter - confirm, Esc - cancel"
		case stateComplete, stateError:
			helpContent += ", Enter - continue"
		}

		helpView = helpStyle.Render(helpContent)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("File Operations"),
		content,
		helpView,
	)
}

// getEncryptionStatusText returns a formatted text for encryption status
func getEncryptionStatusText(info *sops.FileInfo) string {
	if info.Encrypted {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00AA00")).Render("Encrypted")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Render("Not encrypted")
}

// encryptFile encrypts the selected file
func (f *FileEditorView) encryptFile() tea.Cmd {
	return func() tea.Msg {
		recipients := []string{f.recipientInput}

		// Extract filename for result message
		filename := filepath.Base(f.selectedFile)

		// Encrypt file
		err := sops.EncryptFile(f.selectedFile, recipients, true)
		if err != nil {
			return OperationErrorMsg{Error: err}
		}

		return OperationCompleteMsg{
			Message: fmt.Sprintf("Successfully encrypted %s", filename),
		}
	}
}

// decryptFile decrypts the selected file
func (f *FileEditorView) decryptFile() tea.Cmd {
	return func() tea.Msg {
		// Extract filename for result message
		filename := filepath.Base(f.selectedFile)

		// Generate output filename by removing .enc if present
		outputPath := strings.TrimSuffix(f.selectedFile, ".enc")
		if outputPath == f.selectedFile {
			outputPath = f.selectedFile + ".dec"
		}

		// Decrypt file
		err := sops.DecryptFile(f.selectedFile, false, outputPath)
		if err != nil {
			return OperationErrorMsg{Error: err}
		}

		return OperationCompleteMsg{
			Message: fmt.Sprintf("Successfully decrypted %s to %s", filename, filepath.Base(outputPath)),
		}
	}
}

// editFile opens the encrypted file in an editor
func (f *FileEditorView) editFile() tea.Cmd {
	return func() tea.Msg {
		// Extract filename for result message
		filename := filepath.Base(f.selectedFile)

		// Edit file
		err := sops.EditFile(f.selectedFile)
		if err != nil {
			return OperationErrorMsg{Error: err}
		}

		return OperationCompleteMsg{
			Message: fmt.Sprintf("Successfully edited %s", filename),
		}
	}
}

// checkKeyStatus checks if a decrypted key exists
func (f *FileEditorView) checkKeyStatus() tea.Cmd {
	return func() tea.Msg {
		f.hasDecryptedKey = age.IsKeyDecrypted()
		return nil
	}
}

// OperationCompleteMsg is sent when an operation completes successfully
type OperationCompleteMsg struct {
	Message string
}

// OperationErrorMsg is sent when an operation fails
type OperationErrorMsg struct {
	Error error
}

