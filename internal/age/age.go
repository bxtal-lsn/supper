package age

import (
	"bytes"
	"errors"
	"fmt"
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
	cmd := exec.Command("age", "-p", "-o", "-")
	cmd.Stdin = strings.NewReader(key.PrivateKey)

	// Provide passphrase via environment variable to avoid command line exposure
	cmd.Env = append(os.Environ(), fmt.Sprintf("AGE_PASSPHRASE=%s", passphrase))

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to encrypt age key: %w", err)
	}

	return out.Bytes(), nil
}

// DecryptKey decrypts an encrypted age key
func DecryptKey(encryptedKey []byte, passphrase string) (string, error) {
	cmd := exec.Command("age", "-d")
	cmd.Stdin = bytes.NewReader(encryptedKey)

	// Provide passphrase via environment variable to avoid command line exposure
	cmd.Env = append(os.Environ(), fmt.Sprintf("AGE_PASSPHRASE=%s", passphrase))

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to decrypt age key: %w", err)
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
	zeros := make([]byte, 1024)
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
