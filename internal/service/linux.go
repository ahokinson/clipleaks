//go:build linux

package service

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

//go:embed templates/systemd.service.tmpl
var systemdTemplate string

type linuxService struct {
	servicePath string
	serviceName string
}

func newPlatformService() Service {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}
	return &linuxService{
		servicePath: filepath.Join(home, ".config", "systemd", "user", "clipleaks.service"),
		serviceName: "clipleaks.service",
	}
}

func (s *linuxService) Install() error {
	if _, err := os.Stat(s.servicePath); err == nil {
		return fmt.Errorf("%w: %s", ErrServiceAlreadyInstalled, s.servicePath)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("%w: failed to get executable path: %w", ErrServiceInstallFailed, err)
	}

	execPath, err = validateExecutablePath(execPath)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrServiceInstallFailed, err)
	}

	dir := filepath.Dir(s.servicePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("%w: failed to create systemd directory: %w", ErrServiceInstallFailed, err)
	}

	tmpl, err := template.New("systemd").Parse(systemdTemplate)
	if err != nil {
		return fmt.Errorf("%w: failed to parse systemd template: %w", ErrServiceInstallFailed, err)
	}

	data := struct {
		ExecPath string
	}{
		ExecPath: execPath,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("%w: failed to execute systemd template: %w", ErrServiceInstallFailed, err)
	}

	if err := os.WriteFile(s.servicePath, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("%w: failed to write service file: %w", ErrServiceInstallFailed, err)
	}

	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: failed to reload systemd: %w", ErrServiceInstallFailed, err)
	}

	cmd = exec.Command("systemctl", "--user", "enable", s.serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: failed to enable service: %w", ErrServiceInstallFailed, err)
	}

	cmd = exec.Command("systemctl", "--user", "start", s.serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: failed to start service: %w", ErrServiceInstallFailed, err)
	}

	return nil
}

func (s *linuxService) Uninstall() error {
	if _, err := os.Stat(s.servicePath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrServiceNotInstalled, s.servicePath)
	}

	cmd := exec.Command("systemctl", "--user", "stop", s.serviceName)
	_ = cmd.Run()

	cmd = exec.Command("systemctl", "--user", "disable", s.serviceName)
	_ = cmd.Run()

	if err := os.Remove(s.servicePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: failed to remove service file: %w", ErrServiceUninstallFailed, err)
	}

	cmd = exec.Command("systemctl", "--user", "daemon-reload")
	_ = cmd.Run()

	return nil
}

func (s *linuxService) Status() (string, error) {
	if _, err := os.Stat(s.servicePath); errors.Is(err, os.ErrNotExist) {
		return "not installed", nil
	}

	cmd := exec.Command("systemctl", "--user", "is-active", s.serviceName)
	output, err := cmd.Output()
	if err != nil {
		return "installed but not running", nil
	}

	status := string(output)
	if status == "active\n" {
		return "running", nil
	}

	return "installed but not running", nil
}
