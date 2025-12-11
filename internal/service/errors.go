package service

import "errors"

var (
	ErrServiceNotInstalled      = errors.New("service is not installed")
	ErrServiceAlreadyInstalled  = errors.New("service is already installed")
	ErrServiceNotRunning        = errors.New("service is not running")
	ErrServiceAlreadyRunning    = errors.New("service is already running")
	ErrServiceInstallFailed     = errors.New("failed to install service")
	ErrServiceUninstallFailed   = errors.New("failed to uninstall service")
	ErrServiceStatusUnavailable = errors.New("unable to determine service status")
)
