package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateExecutablePath(t *testing.T) {
	dir := t.TempDir()

	exe := filepath.Join(dir, "tool")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	nonExec := filepath.Join(dir, "data")
	if err := os.WriteFile(nonExec, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("valid executable", func(t *testing.T) {
		got, err := validateExecutablePath(exe)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want, _ := filepath.EvalSymlinks(exe)
		if got != want {
			t.Errorf("resolved path = %q, want %q", got, want)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		if _, err := validateExecutablePath(""); err == nil {
			t.Error("expected error for empty path")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := validateExecutablePath(filepath.Join(dir, "nope")); err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("not executable", func(t *testing.T) {
		_, err := validateExecutablePath(nonExec)
		if !errors.Is(err, ErrNotExecutable) {
			t.Fatalf("expected ErrNotExecutable, got: %v", err)
		}
	})

	t.Run("directory is not a regular file", func(t *testing.T) {
		_, err := validateExecutablePath(dir)
		if !errors.Is(err, ErrNotRegularFile) {
			t.Fatalf("expected ErrNotRegularFile, got: %v", err)
		}
	})
}
