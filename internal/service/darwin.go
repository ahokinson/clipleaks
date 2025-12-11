//go:build darwin

package service

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

//go:embed templates/launchd.plist.tmpl
var launchdTemplate string

type darwinService struct {
	plistPath string
	label     string
}

func getServiceIdentifier() string {
	if customName := os.Getenv("CLIPLEAKS_SERVICE_NAME"); customName != "" {
		return customName
	}

	username := os.Getenv("USER")
	if username == "" {
		username = "default"
	}

	hash := sha256.Sum256([]byte(username))
	suffix := hex.EncodeToString(hash[:4])

	return fmt.Sprintf("dev.kingsfoil.clipleaks.agent.%s", suffix)
}

func newPlatformService() Service {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}

	label := getServiceIdentifier()
	plistName := label + ".plist"

	return &darwinService{
		plistPath: filepath.Join(home, "Library", "LaunchAgents", plistName),
		label:     label,
	}
}

func (s *darwinService) Install() error {
	if _, err := os.Stat(s.plistPath); err == nil {
		return fmt.Errorf("%w: %s", ErrServiceAlreadyInstalled, s.plistPath)
	}

	execPath, err := s.getValidatedExecutablePath()
	if err != nil {
		return err
	}

	stdoutLog, stderrLog, err := s.setupLogFiles()
	if err != nil {
		return err
	}

	if err := s.writePlistFile(execPath, stdoutLog, stderrLog); err != nil {
		return err
	}

	return s.loadLaunchAgent()
}

func (s *darwinService) getValidatedExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("%w: failed to get executable path: %w", ErrServiceInstallFailed, err)
	}

	execPath, err = validateExecutablePath(execPath)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrServiceInstallFailed, err)
	}

	return execPath, nil
}

func (s *darwinService) setupLogFiles() (string, string, error) {
	dir := filepath.Dir(s.plistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", fmt.Errorf("%w: failed to create LaunchAgents directory: %w", ErrServiceInstallFailed, err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}
	logDir := filepath.Join(home, ".local", "share", "clipleaks")
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return "", "", fmt.Errorf("%w: failed to create log directory: %w", ErrServiceInstallFailed, err)
	}

	stdoutLog := filepath.Join(logDir, "clipleaks.log")
	stderrLog := filepath.Join(logDir, "clipleaks.error.log")

	for _, logFile := range []string{stdoutLog, stderrLog} {
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			if err := os.WriteFile(logFile, []byte{}, 0600); err != nil {
				return "", "", fmt.Errorf("%w: failed to create log file %s: %w", ErrServiceInstallFailed, logFile, err)
			}
		}
	}

	return stdoutLog, stderrLog, nil
}

func (s *darwinService) writePlistFile(execPath, stdoutLog, stderrLog string) error {
	tmpl, err := template.New("launchd").Parse(launchdTemplate)
	if err != nil {
		return fmt.Errorf("%w: failed to parse plist template: %w", ErrServiceInstallFailed, err)
	}

	data := struct {
		Label     string
		ExecPath  string
		StdoutLog string
		StderrLog string
	}{
		Label:     s.label,
		ExecPath:  execPath,
		StdoutLog: stdoutLog,
		StderrLog: stderrLog,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("%w: failed to execute plist template: %w", ErrServiceInstallFailed, err)
	}

	if err := os.WriteFile(s.plistPath, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("%w: failed to write plist file: %w", ErrServiceInstallFailed, err)
	}

	return nil
}

func (s *darwinService) loadLaunchAgent() error {
	cmd := exec.Command("launchctl", "load", s.plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: failed to load Launch Agent: %w\nOutput: %s", ErrServiceInstallFailed, err, string(output))
	}
	return nil
}

func (s *darwinService) Uninstall() error {
	if _, err := os.Stat(s.plistPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrServiceNotInstalled, s.plistPath)
	}

	cmd := exec.Command("launchctl", "unload", s.plistPath)
	_ = cmd.Run()

	if err := os.Remove(s.plistPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: failed to remove plist file: %w", ErrServiceUninstallFailed, err)
	}

	return nil
}

func (s *darwinService) Status() (string, error) {
	if _, err := os.Stat(s.plistPath); errors.Is(err, os.ErrNotExist) {
		return "not installed", nil
	}

	cmd := exec.Command("launchctl", "list", s.label)
	if err := cmd.Run(); err != nil {
		return "installed but not running", err
	}

	return "running", nil
}
