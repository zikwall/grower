package fileio

import (
	"fmt"
	"os"
	"os/exec"
	"path"
)

// Capture rotate old log file
func capture(dir, file string) (string, error) {
	oldFilepath := path.Join(dir, file)
	if err := check(oldFilepath); err != nil {
		return "", err
	}
	newFilepath := path.Join(dir, logName(file))
	if err := os.Rename(oldFilepath, newFilepath); err != nil {
		return "", fmt.Errorf("failed to capture file: %w", err)
	}
	return newFilepath, nil
}

// Reopen send command to nginx for reopen log file
func reopen() error {
	if err := exec.Command("nginx", "-s", "reopen").Run(); err != nil {
		return fmt.Errorf("failed to reopen nginx: %w", err)
	}
	return nil
}
