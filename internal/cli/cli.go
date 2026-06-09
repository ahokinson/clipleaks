package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"

	"github.com/ahokinson/clipleaks/internal/config"
	"github.com/ahokinson/clipleaks/internal/process"
	"github.com/ahokinson/clipleaks/internal/service"
)

type CLI struct {
	args    []string
	stdout  io.Writer
	stderr  io.Writer
	version VersionInfo
}

type VersionInfo struct {
	Version string
	Commit  string
}

func New(args []string, stdout, stderr io.Writer, version VersionInfo) *CLI {
	return &CLI{
		args:    args,
		stdout:  stdout,
		stderr:  stderr,
		version: version,
	}
}

func (c *CLI) Run() int {
	fs := flag.NewFlagSet("clipleaks", flag.ContinueOnError)
	fs.SetOutput(c.stderr)

	installCmd := fs.Bool("i", false, "Install Clipleaks as a background service")
	uninstallCmd := fs.Bool("u", false, "Uninstall Clipleaks background service")
	statusCmd := fs.Bool("s", false, "Check service status")
	versionCmd := fs.Bool("v", false, "Show version information")

	if err := fs.Parse(c.args); err != nil {
		return 1
	}

	if *versionCmd {
		return c.runVersion()
	}

	svc := service.New()

	if *installCmd {
		return c.runInstall(svc)
	}

	if *uninstallCmd {
		return c.runUninstall(svc)
	}

	if *statusCmd {
		return c.runStatus(svc)
	}

	return c.runMonitor()
}

func (c *CLI) runVersion() int {
	fmt.Fprintf(c.stdout, "Clipleaks %s\n", c.version.Version)
	fmt.Fprintf(c.stdout, "  commit: %s\n", c.version.Commit)
	return 0
}

func (c *CLI) runInstall(svc service.Service) int {
	slog.Info("Installing Clipleaks as a background service")
	if err := svc.Install(); err != nil {
		if errors.Is(err, service.ErrServiceAlreadyInstalled) {
			slog.Warn("Service is already installed")
			return 0
		}
		slog.Error("Failed to install service", "error", err)
		return 1
	}
	slog.Info("Clipleaks installed successfully")
	slog.Info("The service is now running in the background")
	slog.Info("Useful commands: clipleaks -s (check status), clipleaks -u (remove service)")
	return 0
}

func (c *CLI) runUninstall(svc service.Service) int {
	slog.Info("Uninstalling Clipleaks background service")
	if err := svc.Uninstall(); err != nil {
		if errors.Is(err, service.ErrServiceNotInstalled) {
			slog.Warn("Service is not installed")
			return 0
		}
		slog.Error("Failed to uninstall service", "error", err)
		return 1
	}
	slog.Info("Clipleaks uninstalled successfully")
	return 0
}

func (c *CLI) runStatus(svc service.Service) int {
	status, err := svc.Status()
	if err != nil {
		slog.Error("Failed to check status", "error", err)
		return 1
	}
	slog.Info("Clipleaks status", "status", status)
	return 0
}

func (c *CLI) runMonitor() int {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return 1
	}

	slog.Info("Configuration loaded",
		"config_path", config.GetConfigPath(),
		"entropy_threshold", cfg.EntropyThreshold,
		"poll_interval", cfg.PollInterval,
		"max_clipboard_size", *cfg.MaxClipboardSize,
		"pattern_match_size", *cfg.PatternMatchSize,
		"pattern_overlap_size", *cfg.PatternOverlapSize,
		"detection_burst_limit", *cfg.DetectionBurstLimit,
		"detection_rate_limit", *cfg.DetectionRateLimit,
		"disabled_patterns", len(cfg.DisabledPatterns))

	proc := process.New(cfg, c.stdout)

	if err := proc.Run(); err != nil {
		slog.Error("Process failed", "error", err)
		return 1
	}

	return 0
}
