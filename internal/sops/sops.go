package sops

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// FileInfo represents metadata about a SOPS-encrypted file
type FileInfo struct {
	Path       string
	Encrypted  bool
	Recipients []string
}

// EncryptFile encrypts a file using SOPS and age
func EncryptFile(filePath string, ageRecipients []string, inPlace bool) error {
	args := []string{}

	// Add age recipients
	if len(ageRecipients) > 0 {
		recipientArg := "--age=" + strings.Join(ageRecipients, ",")
		args = append(args, recipientArg)
	}

	// Add encrypt flag
	args = append(args, "-e")

	// Add in-place flag if requested
	if inPlace {
		args = append(args, "-i")
	}

	// Add file path
	args = append(args, filePath)

	// Execute SOPS command
	cmd := exec.Command("sops", args...)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to encrypt file: %s - %w", errOut.String(), err)
	}

	return nil
}

// DecryptFile decrypts a file using SOPS
func DecryptFile(filePath string, inPlace bool, outputPath string) error {
	args := []string{"-d"}

	// Add in-place flag if requested
	if inPlace {
		args = append(args, "-i")
	}

	// Add output path if provided
	if outputPath != "" && !inPlace {
		args = append(args, "--output", outputPath)
	}

	// Add file path
	args = append(args, filePath)

	// Execute SOPS command
	cmd := exec.Command("sops", args...)
	var errOut bytes.Buffer
	cmd.Stderr = &errOut

	// If not in-place or specific output, capture stdout
	var out bytes.Buffer
	if !inPlace && outputPath == "" {
		cmd.Stdout = &out
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to decrypt file: %s - %w", errOut.String(), err)
	}

	// If output path is not provided and not in-place, write to stdout
	if !inPlace && outputPath == "" {
		fmt.Print(out.String())
	}

	return nil
}

// EditFile opens a SOPS-encrypted file in an editor
func EditFile(filePath string) error {
	cmd := exec.Command("sops", filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// GetFileInfo retrieves information about a SOPS file
func GetFileInfo(filePath string) (*FileInfo, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Use SOPS to check if the file is encrypted
	cmd := exec.Command("sops", "--output-type", "json", "filestatus", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out

	var info FileInfo
	info.Path = filePath

	if err := cmd.Run(); err != nil {
		// If command fails, assume file is not encrypted
		info.Encrypted = false
		return &info, nil
	}

	// Parse output to determine if file is encrypted
	output := out.String()
	if strings.Contains(output, "\"encrypted\": true") {
		info.Encrypted = true
	}

	// If encrypted, get list of recipients
	if info.Encrypted {
		// This would require parsing the SOPS metadata
		// For now, we'll return an empty list
		info.Recipients = []string{}
	}

	return &info, nil
}

// AddRecipient adds a recipient to an encrypted file
func AddRecipient(filePath string, recipient string) error {
	cmd := exec.Command("sops", "updatekeys", "--age", recipient, filePath)
	var errOut bytes.Buffer
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add recipient: %s - %w", errOut.String(), err)
	}

	return nil
}

// RotateKey rotates the data key in an encrypted file
func RotateKey(filePath string) error {
	cmd := exec.Command("sops", "rotate", "-i", filePath)
	var errOut bytes.Buffer
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to rotate key: %s - %w", errOut.String(), err)
	}

	return nil
}
