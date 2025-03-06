package views

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bxtal-lsn/supper/internal/age"
	"github.com/bxtal-lsn/supper/internal/ui/components"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Key manager states
const (
	stateIdle int = iota
	stateGeneratingKey
	stateInputPassphrase
	stateConfirmPassphrase
	stateDecryptingKey
	stateDeletingKey
)

// Key manager events
type keyGenerated struct {
	keyPair *age.KeyPair
}

type keyDecrypted struct {
	key string
}

type keyDeleted struct{}

// KeyManagerView is the view for managing age keys
type KeyManagerView struct {
	keys               KeyMap
	viewport           viewport.Model
	spinner            spinner.Model
	passphraseInput    *components.PassphraseInput
	width              int
	height             int
	state              int
	keyPair            *age.KeyPair
	encryptedKeyPath   string
	decryptedKeyPath   string
	hasDecryptedKey    bool
	keyDecryptedTime   time.Time
	autoDeleteInterval time.Duration
	err                error
}

// NewKeyManagerView creates a new key manager view
func NewKeyManagerView() *KeyManagerView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &KeyManagerView{
		keys:               DefaultKeyMap(),
		spinner:            s,
		state:              stateIdle,
		encryptedKeyPath:   age.DefaultEncryptedKeyPath(),
		decryptedKeyPath:   age.DefaultKeyPath(),
		autoDeleteInterval: 30 * time.Minute, // Auto-delete decrypted key after 30 minutes
	}
}

// Init initializes the view
func (k *KeyManagerView) Init() tea.Cmd {
	return tea.Batch(
		k.checkKeyStatus(),
		k.spinner.Tick,
	)
}

// Update handles events and updates the model
func (k *KeyManagerView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		k.width = msg.Width
		k.height = msg.Height
		k.viewport = viewport.New(msg.Width, msg.Height-5)
		k.viewport.YPosition = 2

	case tea.KeyMsg:
		// Global key handlers
		switch {
		case key.Matches(msg, k.keys.GenerateKey) && k.state == stateIdle:
			k.state = stateInputPassphrase
			k.passphraseInput = components.NewPassphraseInput("Enter passphrase for new key", true)
			return k, k.passphraseInput.Init()

		case key.Matches(msg, k.keys.DecryptKey) && k.state == stateIdle:
			if _, err := os.Stat(k.encryptedKeyPath); os.IsNotExist(err) {
				k.err = fmt.Errorf("no encrypted key found at %s", k.encryptedKeyPath)
				return k, nil
			}
			k.state = stateDecryptingKey
			k.passphraseInput = components.NewPassphraseInput("Enter passphrase to decrypt key", false)
			return k, k.passphraseInput.Init()

		case key.Matches(msg, k.keys.DeleteKey) && k.state == stateIdle && k.hasDecryptedKey:
			k.state = stateDeletingKey
			return k, k.deleteDecryptedKey()
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		k.spinner, cmd = k.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case keyGenerated:
		k.keyPair = msg.keyPair
		k.state = stateIdle
		cmds = append(cmds, k.checkKeyStatus())

	case keyDecrypted:
		k.state = stateIdle
		k.keyDecryptedTime = time.Now()
		cmds = append(cmds, k.checkKeyStatus())
		// Set a timer to auto-delete the key
		cmds = append(cmds, tea.Tick(k.autoDeleteInterval, func(t time.Time) tea.Msg {
			return tea.KeyMsg{Type: tea.KeyCtrlD}
		}))

	case keyDeleted:
		k.state = stateIdle
		cmds = append(cmds, k.checkKeyStatus())

	case components.PassphraseConfirmedMsg:
		switch k.state {
		case stateInputPassphrase:
			return k, tea.Batch(
				k.generateKey(msg.Passphrase),
				k.spinner.Tick,
			)
		case stateDecryptingKey:
			return k, tea.Batch(
				k.decryptKey(msg.Passphrase),
				k.spinner.Tick,
			)
		}

	case components.PassphraseCancelledMsg:
		k.state = stateIdle
	}

	// Update sub-components
	if k.passphraseInput != nil && (k.state == stateInputPassphrase || k.state == stateDecryptingKey) {
		newModel, cmd := k.passphraseInput.Update(msg)
		if updatedModel, ok := newModel.(*components.PassphraseInput); ok {
			k.passphraseInput = updatedModel
		}
		cmds = append(cmds, cmd)
	}

	return k, tea.Batch(cmds...)
}

// View renders the view
func (k *KeyManagerView) View() string {
	var content string

	switch k.state {
	case stateIdle:
		content = k.renderIdleState()
	case stateGeneratingKey:
		content = fmt.Sprintf("%s Generating key...", k.spinner.View())
	case stateInputPassphrase, stateDecryptingKey:
		if k.passphraseInput != nil {
			content = k.passphraseInput.View()
		}
	case stateDeletingKey:
		content = fmt.Sprintf("%s Securely deleting key...", k.spinner.View())
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#1E88E5")).Padding(0, 1).Render("Age Key Management"),
		content,
	)
}

// renderIdleState renders the idle state view
func (k *KeyManagerView) renderIdleState() string {
	var content string

	keyStyle := lipgloss.NewStyle().Width(60).Border(lipgloss.RoundedBorder()).Padding(1)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00AA00"))

	if k.err != nil {
		content += errorStyle.Render(fmt.Sprintf("Error: %s", k.err)) + "\n\n"
	}

	if k.hasDecryptedKey {
		elapsedTime := time.Since(k.keyDecryptedTime)
		remainingTime := k.autoDeleteInterval - elapsedTime
		if remainingTime < 0 {
			remainingTime = 0
		}

		content += infoStyle.Render("Key Status: Decrypted") + "\n"
		content += fmt.Sprintf("Decrypted Key Path: %s\n", k.decryptedKeyPath)
		content += fmt.Sprintf("Auto-Delete In: %s\n\n", remainingTime.Round(time.Second))
		content += fmt.Sprintf("Public Key: %s\n\n", k.keyPair.PublicKey)
		content += "Press 'x' to securely delete the decrypted key now.\n\n"
	} else {
		content += "Key Status: " + errorStyle.Render("Not Decrypted") + "\n\n"

		if _, err := os.Stat(k.encryptedKeyPath); err == nil {
			content += fmt.Sprintf("Encrypted Key Path: %s\n", k.encryptedKeyPath)
			content += "Press 'd' to decrypt the key.\n\n"
		} else {
			content += "No encrypted key found.\n"
			content += "Press 'g' to generate a new key.\n\n"
		}
	}

	return keyStyle.Render(content)
}

// checkKeyStatus checks if a decrypted key exists
func (k *KeyManagerView) checkKeyStatus() tea.Cmd {
	return func() tea.Msg {
		k.hasDecryptedKey = age.IsKeyDecrypted()

		// If key is decrypted, read the key to get public key
		if k.hasDecryptedKey && k.keyPair == nil {
			data, err := os.ReadFile(k.decryptedKeyPath)
			if err == nil {
				// This is a simplification - proper parsing would be more complex
				privateKey := string(data)
				// Extract public key from private key (would require proper age key parsing)
				publicKey := "age1..." // Placeholder
				k.keyPair = &age.KeyPair{
					PrivateKey:  privateKey,
					PublicKey:   publicKey,
					IsEncrypted: false,
				}
			}
		}

		return nil
	}
}

// generateKey generates a new age key
func (k *KeyManagerView) generateKey(passphrase string) tea.Cmd {
	k.state = stateGeneratingKey
	k.err = nil

	return func() tea.Msg {
		// Generate key
		keyPair, err := age.GenerateKey()
		if err != nil {
			k.err = fmt.Errorf("failed to generate key: %w", err)
			return keyGenerated{keyPair: nil}
		}

		// Encrypt with passphrase
		encryptedKey, err := age.EncryptKey(keyPair, passphrase)
		if err != nil {
			k.err = fmt.Errorf("failed to encrypt key: %w", err)
			return keyGenerated{keyPair: nil}
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(k.encryptedKeyPath), 0o700); err != nil {
			k.err = fmt.Errorf("failed to create directory: %w", err)
			return keyGenerated{keyPair: nil}
		}

		// Save encrypted key
		if err := age.SaveEncryptedKey(encryptedKey, k.encryptedKeyPath); err != nil {
			k.err = fmt.Errorf("failed to save encrypted key: %w", err)
			return keyGenerated{keyPair: nil}
		}

		// Save decrypted key
		if err := age.SaveKey(keyPair, k.decryptedKeyPath); err != nil {
			k.err = fmt.Errorf("failed to save decrypted key: %w", err)
			return keyGenerated{keyPair: nil}
		}

		return keyGenerated{keyPair: keyPair}
	}
}

// decryptKey decrypts an age key
func (k *KeyManagerView) decryptKey(passphrase string) tea.Cmd {
	k.state = stateDecryptingKey
	k.err = nil

	return func() tea.Msg {
		// Load encrypted key
		encryptedKey, err := age.LoadEncryptedKey(k.encryptedKeyPath)
		if err != nil {
			k.err = fmt.Errorf("failed to load encrypted key: %w", err)
			return keyDecrypted{key: ""}
		}

		// Decrypt key
		decryptedKey, err := age.DecryptKey(encryptedKey, passphrase)
		if err != nil {
			k.err = fmt.Errorf("failed to decrypt key: %w", err)
			return keyDecrypted{key: ""}
		}

		// Save decrypted key
		if err := os.MkdirAll(filepath.Dir(k.decryptedKeyPath), 0o700); err != nil {
			k.err = fmt.Errorf("failed to create directory: %w", err)
			return keyDecrypted{key: ""}
		}

		if err := os.WriteFile(k.decryptedKeyPath, []byte(decryptedKey), 0o600); err != nil {
			k.err = fmt.Errorf("failed to save decrypted key: %w", err)
			return keyDecrypted{key: ""}
		}

		// This is a simplification - proper parsing would be more complex
		// Extract public key from private key (would require proper age key parsing)
		publicKey := "age1..." // Placeholder
		k.keyPair = &age.KeyPair{
			PrivateKey:  decryptedKey,
			PublicKey:   publicKey,
			IsEncrypted: false,
		}

		return keyDecrypted{key: decryptedKey}
	}
}

// deleteDecryptedKey securely deletes the decrypted key
func (k *KeyManagerView) deleteDecryptedKey() tea.Cmd {
	k.err = nil

	return func() tea.Msg {
		if err := age.SecurelyDeleteKey(k.decryptedKeyPath); err != nil {
			k.err = fmt.Errorf("failed to delete key: %w", err)
		}
		k.keyPair = nil
		return keyDeleted{}
	}
}

