// Package rollbackinspector checks retained rollback files without executing them.
package rollbackinspector

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var rollbackFileName = regexp.MustCompile(`^(.+)_([0-9]+)_([[:alnum:]]{8})\.py$`)

// Inspect reports whether a retained rollback script exists for an attempt.
// It examines directory entries without following symlinks or executing files.
func Inspect(root, attemptID string) (bool, error) {
	if root == "" || attemptID == "" || strings.ContainsAny(attemptID, `/\\`) {
		return false, fmt.Errorf("rollback inspector requires a directory and attempt ID")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return false, fmt.Errorf("read rollback directory: %w", err)
	}
	for _, directory := range entries {
		if !directory.IsDir() || !attemptDirectory(directory.Name(), attemptID) {
			continue
		}
		files, err := os.ReadDir(filepath.Join(root, directory.Name()))
		if err != nil {
			return false, fmt.Errorf("read attempt directory: %w", err)
		}
		for _, file := range files {
			if !rollbackFileName.MatchString(file.Name()) {
				continue
			}
			info, err := os.Lstat(filepath.Join(root, directory.Name(), file.Name()))
			if err != nil {
				return false, fmt.Errorf("inspect rollback file: %w", err)
			}
			if info.Mode().IsRegular() {
				return true, nil
			}
		}
	}
	return false, nil
}

func attemptDirectory(name, attemptID string) bool {
	prefix, id, ok := strings.Cut(name, "-")
	if !ok || id != attemptID || prefix == "" {
		return false
	}
	for _, char := range prefix {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// Run writes the probe result to Kubernetes' termination message file.
func Run(root, attemptID, messagePath string) error {
	found, err := Inspect(root, attemptID)
	if err != nil {
		return err
	}
	result := "absent"
	if found {
		result = "available"
	}
	if err := os.WriteFile(messagePath, []byte(result), 0600); err != nil {
		return fmt.Errorf("write rollback inspection result: %w", err)
	}
	return nil
}
