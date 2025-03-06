package age

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// KeyPair represents an age key pair
type KeyPair struct {
	PrivateKey  string
	PublicKey   string
	IsEncrypted bool
}

// DefaultKeyPath returns the default path for the age key
func DefaultKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "sops", "age", "keys.txt")
}

// DefaultEncryptedKeyPath returns the default path for the encrypted age key
func DefaultEncryptedKeyPath() string {
	return DefaultKeyPath() + ".encrypted"
}

// GenerateKey generates a new age key pair
func GenerateKey() (*KeyPair, error) {
	cmd := exec.Command("age-keygen")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to generate age key: %w", err)
	}

	output := out.String()
	lines := strings.Split(output, "\n")

	// Extract public key from the output
	var publicKey string
	var privateKey string
	for _, line := range lines {
		if strings.HasPrefix(line, "# public key: ") {
			publicKey = strings.TrimPrefix(line, "# public key: ")
		} else if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			privateKey = line
		}
	}

	if publicKey == "" || privateKey == "" {
		return nil, errors.New("failed to parse age key output")
	}

	return &KeyPair{
		PrivateKey:  privateKey,
		PublicKey:   publicKey,
		IsEncrypted: false,
	}, nil
}

// EncryptKey encrypts an age key with a passphrase
func EncryptKey(key *KeyPair, passphrase string) ([]byte, error) {
	// Create a temporary file to write the private key
	tmpFile, err := os.CreateTemp("", "age-key-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file

	// Write the private key to the temp file
	if _, err := tmpFile.WriteString(key.PrivateKey); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write private key to temporary file: %w", err)
	}
	tmpFile.Close()

	// Set up command to encrypt using stdin for passphrase
	cmd := exec.Command("age", "-p", "-o", "-", tmpPath)

	// Connect passphrase to stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start the command before writing to stdin
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start age command: %w", err)
	}

	// Write passphrase to stdin twice (age requires confirmation)
	if _, err := io.WriteString(stdin, passphrase+"\n"+passphrase+"\n"); err != nil {
		return nil, fmt.Errorf("failed to write passphrase: %w", err)
	}
	stdin.Close()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("failed to encrypt key: %s - %w", errOut.String(), err)
	}

	return out.Bytes(), nil
}

// DecryptKey decrypts an encrypted age key
func DecryptKey(encryptedKey []byte, passphrase string) (string, error) {
	// Create a temporary file for the encrypted key
	tmpFile, err := os.CreateTemp("", "age-encrypted-*.key")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file

	// Write encrypted key to temp file
	if _, err := tmpFile.Write(encryptedKey); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tmpFile.Close()

	// Set up command to use stdin for passphrase instead of env var
	cmd := exec.Command("age", "-d", "-i", tmpPath)

	// Connect passphrase to stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start the command before writing to stdin
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start age command: %w", err)
	}

	// Write passphrase to stdin and close
	if _, err := io.WriteString(stdin, passphrase+"\n"); err != nil {
		return "", fmt.Errorf("failed to write passphrase: %w", err)
	}
	stdin.Close()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("failed to decrypt key: %s - %w", errOut.String(), err)
	}

	return out.String(), nil
}

// SaveKey saves an age key to the specified file
func SaveKey(key *KeyPair, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(key.PrivateKey), 0o600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// SaveEncryptedKey saves an encrypted age key to the specified file
func SaveEncryptedKey(encryptedKey []byte, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, encryptedKey, 0o600); err != nil {
		return fmt.Errorf("failed to write encrypted key file: %w", err)
	}

	return nil
}

// LoadEncryptedKey loads an encrypted age key from the specified file
func LoadEncryptedKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted key file: %w", err)
	}

	return data, nil
}

// SecurelyDeleteKey securely deletes the decrypted key file
func SecurelyDeleteKey(path string) error {
	// Check if shred is available
	_, err := exec.LookPath("shred")
	if err == nil {
		cmd := exec.Command("shred", "-u", path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to shred key file: %w", err)
		}
		return nil
	}

	// Fallback: overwrite with zeros before deleting
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat key file: %w", err)
	}

	file, err := os.OpenFile(path, os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open key file for overwriting: %w", err)
	}
	defer file.Close()

	// Overwrite file with zeros
	size := fileInfo.Size()
	zeros := make([]byte, 1024)
	for written := int64(0); written < size; {
		n, err := file.Write(zeros)
		if err != nil {
			return fmt.Errorf("failed to overwrite key file: %w", err)
		}
		written += int64(n)
	}

	// Finally, remove the file
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove key file: %w", err)
	}

	return nil
}

// IsKeyDecrypted checks if the age key is decrypted (exists on disk)
func IsKeyDecrypted() bool {
	_, err := os.Stat(DefaultKeyPath())
	return err == nil
}

