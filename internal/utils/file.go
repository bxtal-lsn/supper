package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// FileExists checks if a file exists and is not a directory
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// EnsureDir ensures a directory exists, creating it if necessary
func EnsureDir(path string) error {
	if DirExists(path) {
		return nil
	}
	return os.MkdirAll(path, 0o700)
}

// SecurelyDeleteFile securely deletes a file
func SecurelyDeleteFile(path string) error {
	// Check if shred is available
	_, err := exec.LookPath("shred")
	if err == nil {
		cmd := exec.Command("shred", "-u", path)
		return cmd.Run()
	}

	// Fallback to overwriting with zeros
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	size := fileInfo.Size()
	zeros := make([]byte, 4096)

	for written := int64(0); written < size; {
		n, err := file.Write(zeros)
		if err != nil {
			return err
		}
		written += int64(n)
	}

	// Finally, remove the file
	return os.Remove(path)
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := EnsureDir(dstDir); err != nil {
		return err
	}

	// Write destination file
	return os.WriteFile(dst, data, 0o600)
}

// GetFileSize returns the size of a file in a human-readable format
func GetFileSize(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	size := info.Size()

	units := []string{"B", "KB", "MB", "GB", "TB"}
	unitIndex := 0
	sizef := float64(size)

	for sizef >= 1024 && unitIndex < len(units)-1 {
		sizef /= 1024
		unitIndex++
	}

	return fmt.Sprintf("%.1f %s", sizef, units[unitIndex]), nil
}
