package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/ahokinson/clipleaks/internal/service"
)

// fakeService is a controllable service.Service for exercising CLI dispatch.
type fakeService struct {
	installErr   error
	uninstallErr error
	status       string
	statusErr    error
}

func (f *fakeService) Install() error          { return f.installErr }
func (f *fakeService) Uninstall() error        { return f.uninstallErr }
func (f *fakeService) Status() (string, error) { return f.status, f.statusErr }

func newCLI(args ...string) (*CLI, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	c := New(args, &out, &errBuf, VersionInfo{Version: "1.2.3", Commit: "abc123"})
	return c, &out, &errBuf
}

func TestRunVersion(t *testing.T) {
	c, out, _ := newCLI("-v")
	if code := c.Run(); code != 0 {
		t.Fatalf("version command exit code = %d, want 0", code)
	}
	s := out.String()
	if !strings.Contains(s, "1.2.3") || !strings.Contains(s, "abc123") {
		t.Errorf("version output missing version/commit: %q", s)
	}
}

func TestRun_UnknownFlagReturnsError(t *testing.T) {
	c, _, _ := newCLI("--definitely-not-a-flag")
	if code := c.Run(); code != 1 {
		t.Fatalf("unknown flag exit code = %d, want 1", code)
	}
}

func TestRunInstall(t *testing.T) {
	tests := []struct {
		name string
		svc  *fakeService
		want int
	}{
		{"success", &fakeService{}, 0},
		{"already installed is treated as success", &fakeService{installErr: service.ErrServiceAlreadyInstalled}, 0},
		{"failure", &fakeService{installErr: errors.New("boom")}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, _ := newCLI()
			if code := c.runInstall(tt.svc); code != tt.want {
				t.Fatalf("runInstall exit code = %d, want %d", code, tt.want)
			}
		})
	}
}

func TestRunUninstall(t *testing.T) {
	tests := []struct {
		name string
		svc  *fakeService
		want int
	}{
		{"success", &fakeService{}, 0},
		{"not installed is treated as success", &fakeService{uninstallErr: service.ErrServiceNotInstalled}, 0},
		{"failure", &fakeService{uninstallErr: errors.New("boom")}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, _ := newCLI()
			if code := c.runUninstall(tt.svc); code != tt.want {
				t.Fatalf("runUninstall exit code = %d, want %d", code, tt.want)
			}
		})
	}
}

func TestRunStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, _, _ := newCLI()
		if code := c.runStatus(&fakeService{status: "running"}); code != 0 {
			t.Fatalf("runStatus exit code = %d, want 0", code)
		}
	})
	t.Run("failure", func(t *testing.T) {
		c, _, _ := newCLI()
		if code := c.runStatus(&fakeService{statusErr: errors.New("boom")}); code != 1 {
			t.Fatalf("runStatus exit code = %d, want 1", code)
		}
	})
}
