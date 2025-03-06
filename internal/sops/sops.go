package sops

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/bxtal-lsn/supper/internal/errors"
	"github.com/bxtal-lsn/supper/internal/recovery"
)

// FileInfo represents metadata about a SOPS-encrypted file
type FileInfo struct {
	Path       string
	Encrypted  bool
	Recipients []string
}

// Common SOPS error patterns for better error detection
var (
	errFailedToDecrypt      = regexp.MustCompile(`(?i)failed to decrypt`)
	errKeyNotFound          = regexp.MustCompile(`(?i)no key.*found`)
	errFileAlreadyEncrypt   = regexp.MustCompile(`(?i)already encrypted`)
	errNoRegexMatch         = regexp.MustCompile(`(?i)no regex match`)
	errMissingConfiguration = regexp.MustCompile(`(?i)could not find sops configuration`)
)

// ParseSOPSError analyzes SOPS error messages to return better structured errors
func ParseSOPSError(cmdErr error, stderr string) error {
	if cmdErr == nil {
		return nil
	}

	switch {
	case errFailedToDecrypt.MatchString(stderr):
		return errors.New(errors.TypeSecurity, "Failed to decrypt file (incorrect key or corrupted file)")
	case errKeyNotFound.MatchString(stderr):
		return errors.New(errors.TypeSecurity, "No suitable decryption key found")
	case errFileAlreadyEncrypt.MatchString(stderr):
		return errors.New(errors.TypeFileOperation, "File is already encrypted")
	case errNoRegexMatch.MatchString(stderr):
		return errors.New(errors.TypeConfig, "SOPS regex pattern did not match any values")
	case errMissingConfiguration.MatchString(stderr):
		return errors.New(errors.TypeConfig, "Missing SOPS configuration (.sops.yaml)")
	default:
		return errors.Wrap(cmdErr, errors.TypeGeneral, "SOPS operation failed").WithData("details", stderr)
	}
}

// EncryptFile encrypts a file using SOPS and age
func EncryptFile(filePath string, ageRecipients []string, inPlace bool) error {
	// Prepare for operation with backup
	tm := recovery.NewTransactionManager()
	if err := tm.Begin(filePath); err != nil {
		return err
	}

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
		// Use recovery mechanism to restore original file
		if rollbackErr := tm.Rollback(); rollbackErr != nil {
			// Both encryption and rollback failed
			return errors.Wrap(err, errors.TypeFileOperation,
				"Failed to encrypt file and rollback also failed").
				WithData("stderr", errOut.String()).
				WithData("rollbackError", rollbackErr.Error())
		}

		// Return parsed error
		return ParseSOPSError(err, errOut.String())
	}

	// Commit the operation (clear backups)
	tm.Commit()
	return nil
}

// DecryptFile decrypts a file using SOPS
func DecryptFile(filePath string, inPlace bool, outputPath string) error {
	// Prepare for operation with backup if modifying in-place
	tm := recovery.NewTransactionManager()
	if inPlace {
		if err := tm.Begin(filePath); err != nil {
			return err
		}
	}

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
		// If in-place operation, rollback
		if inPlace {
			if rollbackErr := tm.Rollback(); rollbackErr != nil {
				return errors.Wrap(err, errors.TypeFileOperation,
					"Failed to decrypt file and rollback also failed").
					WithData("stderr", errOut.String()).
					WithData("rollbackError", rollbackErr.Error())
			}
		}

		return ParseSOPSError(err, errOut.String())
	}

	// If in-place, commit the operation
	if inPlace {
		tm.Commit()
	}

	// If output path is not provided and not in-place, write to stdout
	if !inPlace && outputPath == "" {
		fmt.Print(out.String())
	}

	return nil
}

// EditFile opens a SOPS-encrypted file in an editor
func EditFile(filePath string) error {
	// Create backup before editing
	tm := recovery.NewTransactionManager()
	if err := tm.Begin(filePath); err != nil {
		return err
	}

	cmd := exec.Command("sops", filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// If editing fails, we'll ask if the user wants to restore from backup
		return errors.Wrap(err, errors.TypeFileOperation,
			"Failed to edit file").WithData("path", filePath)
	}

	// Editing was successful, commit the transaction
	tm.Commit()
	return nil
}

// GetFileInfo retrieves information about a SOPS file
func GetFileInfo(filePath string) (*FileInfo, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, errors.Wrap(err, errors.TypeFileOperation,
			"File does not exist").WithData("path", filePath)
	}

	// Use SOPS to check if the file is encrypted
	cmd := exec.Command("sops", "--output-type", "json", "filestatus", filePath)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	var info FileInfo
	info.Path = filePath

	if err := cmd.Run(); err != nil {
		// If command fails, check the error
		if errOut.String() != "" {
			// If there's an error message but it's not about encryption status
			// then return the error
			if !strings.Contains(errOut.String(), "not an encrypted file") {
				return nil, ParseSOPSError(err, errOut.String())
			}
		}

		// Otherwise assume file is not encrypted
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
		info.Recipients = extractRecipients(output)
	}

	return &info, nil
}

// extractRecipients parses the SOPS filestatus output to extract recipients
func extractRecipients(output string) []string {
	var recipients []string

	// Simple regex to find age recipient patterns
	recipientPattern := regexp.MustCompile(`(?i)"recipient":\s*"([^"]+)"`)
	matches := recipientPattern.FindAllStringSubmatch(output, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			recipients = append(recipients, match[1])
		}
	}

	return recipients
}

// AddRecipient adds a recipient to an encrypted file
func AddRecipient(filePath string, recipient string) error {
	// Create backup before modifying
	tm := recovery.NewTransactionManager()
	if err := tm.Begin(filePath); err != nil {
		return err
	}

	cmd := exec.Command("sops", "updatekeys", "--age", recipient, filePath)
	var errOut bytes.Buffer
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		// Rollback if operation fails
		if rollbackErr := tm.Rollback(); rollbackErr != nil {
			return errors.Wrap(err, errors.TypeFileOperation,
				"Failed to add recipient and rollback also failed").
				WithData("stderr", errOut.String()).
				WithData("rollbackError", rollbackErr.Error())
		}

		return ParseSOPSError(err, errOut.String())
	}

	// Operation succeeded, commit
	tm.Commit()
	return nil
}

// RotateKey rotates the data key in an encrypted file
func RotateKey(filePath string) error {
	// Create backup before rotating keys
	tm := recovery.NewTransactionManager()
	if err := tm.Begin(filePath); err != nil {
		return err
	}

	cmd := exec.Command("sops", "rotate", "-i", filePath)
	var errOut bytes.Buffer
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		// Rollback if operation fails
		if rollbackErr := tm.Rollback(); rollbackErr != nil {
			return errors.Wrap(err, errors.TypeFileOperation,
				"Failed to rotate key and rollback also failed").
				WithData("stderr", errOut.String()).
				WithData("rollbackError", rollbackErr.Error())
		}

		return ParseSOPSError(err, errOut.String())
	}

	// Operation succeeded, commit
	tm.Commit()
	return nil
}

