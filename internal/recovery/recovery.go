// internal/recovery/recovery.go
package recovery

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bxtal-lsn/supper/internal/errors"
	"github.com/bxtal-lsn/supper/internal/utils"
)

// BackupManager handles automatic backups and recovery
type BackupManager struct {
	BackupDir  string
	MaxBackups int
}

// NewBackupManager creates a new backup manager
func NewBackupManager(backupDir string) *BackupManager {
	if backupDir == "" {
		// Use default directory in user's config
		configDir, err := os.UserConfigDir()
		if err == nil {
			backupDir = filepath.Join(configDir, "supper", "backups")
		} else {
			// Fall back to temporary directory
			backupDir = filepath.Join(os.TempDir(), "supper-backups")
		}
	}

	return &BackupManager{
		BackupDir:  backupDir,
		MaxBackups: 5, // Keep last 5 backups by default
	}
}

// BackupFile creates a backup of a file before modification
func (bm *BackupManager) BackupFile(filePath string) (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(bm.BackupDir, 0o700); err != nil {
		return "", errors.Wrap(err, errors.TypeFileOperation,
			"Failed to create backup directory")
	}

	// Check if original file exists
	if !utils.FileExists(filePath) {
		return "", errors.New(errors.TypeFileOperation,
			"Cannot backup non-existent file").WithData("path", filePath)
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	fileName := filepath.Base(filePath)
	backupPath := filepath.Join(bm.BackupDir, fmt.Sprintf("%s-%s.bak", fileName, timestamp))

	// Copy the file
	if err := utils.CopyFile(filePath, backupPath); err != nil {
		return "", errors.Wrap(err, errors.TypeFileOperation,
			"Failed to create backup").WithData("source", filePath).WithData("destination", backupPath)
	}

	// Clean up old backups
	bm.cleanupOldBackups(fileName)

	return backupPath, nil
}

// RestoreFromBackup restores a file from its most recent backup
func (bm *BackupManager) RestoreFromBackup(filePath string) (string, error) {
	// Get the most recent backup
	fileName := filepath.Base(filePath)
	backupFiles, err := bm.findBackups(fileName)
	if err != nil {
		return "", err
	}

	if len(backupFiles) == 0 {
		return "", errors.New(errors.TypeFileOperation,
			"No backups found for file").WithData("file", fileName)
	}

	// Most recent backup is the last one (due to sorting by name/date)
	mostRecentBackup := backupFiles[len(backupFiles)-1]
	backupPath := filepath.Join(bm.BackupDir, mostRecentBackup)

	// Restore the file
	if err := utils.CopyFile(backupPath, filePath); err != nil {
		return "", errors.Wrap(err, errors.TypeFileOperation,
			"Failed to restore from backup").WithData("backup", backupPath).WithData("destination", filePath)
	}

	return backupPath, nil
}

// findBackups returns a list of backup files for a given filename, sorted by date
func (bm *BackupManager) findBackups(fileName string) ([]string, error) {
	// Ensure backup directory exists
	if !utils.DirExists(bm.BackupDir) {
		return []string{}, nil
	}

	// Get all files in the backup directory
	files, err := os.ReadDir(bm.BackupDir)
	if err != nil {
		return nil, errors.Wrap(err, errors.TypeFileOperation,
			"Failed to read backup directory").WithData("directory", bm.BackupDir)
	}

	// Filter and sort backup files
	prefix := fileName + "-"
	suffix := ".bak"
	var backups []string

	for _, file := range files {
		name := file.Name()
		if !file.IsDir() && len(name) > len(prefix)+len(suffix) &&
			name[:len(prefix)] == prefix && name[len(name)-len(suffix):] == suffix {
			backups = append(backups, name)
		}
	}

	return backups, nil
}

// cleanupOldBackups removes old backups exceeding the maximum number
func (bm *BackupManager) cleanupOldBackups(fileName string) error {
	backups, err := bm.findBackups(fileName)
	if err != nil {
		return err
	}

	// If we have more backups than the maximum allowed, remove the oldest ones
	if len(backups) > bm.MaxBackups {
		// Delete oldest backups (those at the beginning of the slice)
		for i := 0; i < len(backups)-bm.MaxBackups; i++ {
			backupPath := filepath.Join(bm.BackupDir, backups[i])
			if err := os.Remove(backupPath); err != nil {
				// Just log the error but continue
				fmt.Fprintf(os.Stderr, "Failed to delete old backup %s: %v\n", backupPath, err)
			}
		}
	}

	return nil
}

// TransactionManager handles file operations with backup and rollback
type TransactionManager struct {
	backupManager *BackupManager
	backupPaths   map[string]string
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager() *TransactionManager {
	return &TransactionManager{
		backupManager: NewBackupManager(""),
		backupPaths:   make(map[string]string),
	}
}

// Begin starts a new transaction by backing up files
func (tm *TransactionManager) Begin(filePaths ...string) error {
	tm.backupPaths = make(map[string]string)

	for _, path := range filePaths {
		// Skip non-existent files
		if !utils.FileExists(path) {
			continue
		}

		// Create backup
		backupPath, err := tm.backupManager.BackupFile(path)
		if err != nil {
			// If backup fails, attempt to roll back what we've done so far
			tm.Rollback()
			return err
		}

		tm.backupPaths[path] = backupPath
	}

	return nil
}

// Commit finalizes the transaction
func (tm *TransactionManager) Commit() {
	tm.backupPaths = make(map[string]string)
}

// Rollback restores files from backups
func (tm *TransactionManager) Rollback() error {
	var lastErr error

	for path, backupPath := range tm.backupPaths {
		if utils.FileExists(backupPath) {
			if err := utils.CopyFile(backupPath, path); err != nil {
				lastErr = errors.Wrap(err, errors.TypeFileOperation,
					"Failed to restore file during rollback").WithData("path", path)
			}
		}
	}

	return lastErr
}

