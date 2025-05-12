package apptest

import (
	"os"
	"path/filepath"
)

// findGitDirectory searches for a .git directory starting from the current directory
// and traversing upwards until either finding it or reaching the $HOME directory
func findGitDirectory() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	currentDir, err = filepath.Abs(currentDir)
	if err != nil {
		return "", err
	}
	homeDir, err = filepath.Abs(homeDir)
	if err != nil {
		return "", err
	}

	// Start traversing upwards from the current directory
	for {
		// Check if we're at or beyond the home directory
		if currentDir == homeDir {
			return "", nil // Reached home directory without finding .git
		}

		// Check if the .git directory exists in the current directory
		gitDir := filepath.Join(currentDir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return currentDir, nil // Found .git directory
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)

		// If we can't go up further, stop
		if parentDir == currentDir {
			return "", nil
		}

		currentDir = parentDir
	}
}
