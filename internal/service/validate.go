package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrNotRegularFile = errors.New("executable is not a regular file")
	ErrNotExecutable  = errors.New("file is not executable")
	ErrSymlinkLoop    = errors.New("symlink loop detected")
)

func validateExecutablePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("executable path cannot be empty")
	}

	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("executable does not exist: %w", err)
		}
		if errors.Is(err, filepath.ErrBadPattern) {
			return "", fmt.Errorf("%w: %w", ErrSymlinkLoop, err)
		}
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}

	info, err := os.Stat(realPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat executable: %w", err)
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: %s", ErrNotRegularFile, realPath)
	}

	if info.Mode().Perm()&0111 == 0 {
		return "", fmt.Errorf("%w: %s", ErrNotExecutable, realPath)
	}

	return realPath, nil
}
