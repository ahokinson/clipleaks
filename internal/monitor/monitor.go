package monitor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ahokinson/clipleaks/internal/clipboard"
	"github.com/ahokinson/clipleaks/internal/config"
	"github.com/ahokinson/clipleaks/internal/detector"
	"github.com/ahokinson/clipleaks/internal/logger"
	"github.com/ahokinson/clipleaks/internal/monitor/limit"
	"github.com/ahokinson/clipleaks/internal/notification"
)

const (
	DefaultNotificationRateLimit     = 30 * time.Second       // Prevent notification spam for repeated secrets
	DefaultClipboardDebounceInterval = 100 * time.Millisecond // Ignore rapid clipboard changes (copy/paste operations)
	DefaultDetectionBurstLimit       = 10                     // Allow bursts of up to 10 rapid scans
	DefaultDetectionRateLimit        = 2.0                    // Average 2 scans per second
	ErrorChannelBuffer               = 2                      // Buffer errors from monitor and signal handler
)

const (
	NotificationTitleStartup       = "Clipleaks Started"
	NotificationMessageStartup     = "Clipboard monitoring is now active"
	NotificationTitleSecurityAlert = "Security Alert"
)

type Monitor struct {
	cfg      *config.Config
	detector detector.Scanner
	notifier notification.Notifier
	monitor  clipboard.Monitor
	stdout   io.Writer
	limiter  *limit.Limiter

	initOnce sync.Once
	initErr  error

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	errCh  chan error
}

func New(cfg *config.Config, stdout io.Writer) *Monitor {
	var detectorOpts []detector.Option
	if len(cfg.DisabledPatterns) > 0 {
		detectorOpts = append(detectorOpts, detector.WithDisabledPatterns(cfg.DisabledPatterns...))
	}
	detectorOpts = append(detectorOpts, detector.WithPatternMatchSize(*cfg.PatternMatchSize))
	detectorOpts = append(detectorOpts, detector.WithPatternOverlapSize(*cfg.PatternOverlapSize))
	det := detector.New(detectorOpts...)

	notifier := notification.New(notification.WithRateLimit(DefaultNotificationRateLimit))

	mon := clipboard.New(
		clipboard.WithMaxSize(*cfg.MaxClipboardSize),
		clipboard.WithPollInterval(time.Duration(cfg.PollInterval)*time.Millisecond),
		clipboard.WithDebounceInterval(DefaultClipboardDebounceInterval),
	)

	return &Monitor{
		cfg:      cfg,
		detector: det,
		notifier: notifier,
		monitor:  mon,
		stdout:   stdout,
		limiter:  limit.New(float64(*cfg.DetectionBurstLimit), *cfg.DetectionRateLimit),
		errCh:    make(chan error, ErrorChannelBuffer),
	}
}

func (m *Monitor) Run() error {
	return m.RunWithContext(context.Background())
}

func (m *Monitor) RunWithContext(ctx context.Context) error {
	m.initOnce.Do(func() {
		m.initErr = m.initialize()
	})
	if m.initErr != nil {
		return fmt.Errorf("failed to initialize monitor: %w", m.initErr)
	}

	m.ctx, m.cancel = context.WithCancel(ctx)
	defer func() { _ = m.Close() }()

	m.wg.Add(1)
	go m.handleSignals()

	m.wg.Add(1)
	go m.runMonitoring()

	if err := m.notifier.Notify(NotificationTitleStartup, NotificationMessageStartup); err != nil {
		slog.Error("Notification system is unavailable - this is required for security alerts", "error", err)
		m.cancel()
		m.wg.Wait()
		return fmt.Errorf("notification system unavailable: %w", err)
	}

	slog.Info("Clipleaks is now monitoring your clipboard - Press Ctrl+C to stop")

	err := <-m.errCh

	m.cancel()

	m.wg.Wait()

	select {
	case additionalErr := <-m.errCh:
		if err == nil || errors.Is(err, context.Canceled) {
			err = additionalErr
		}
	default:
	}

	slog.Info("Clipleaks stopped")

	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func (m *Monitor) Close() error {
	if m.cancel != nil {
		m.cancel()
	}
	return nil
}

func (m *Monitor) initialize() error {
	logger.Setup(m.stdout, slog.LevelInfo, false)
	return nil
}

func (m *Monitor) handleSignals() {
	defer m.wg.Done()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case sig := <-sigChan:
		slog.Info("Received shutdown signal", "signal", sig)
		m.errCh <- context.Canceled
	case <-m.ctx.Done():
		m.errCh <- m.ctx.Err()
	}
}

func (m *Monitor) runMonitoring() {
	defer m.wg.Done()

	contentChan, err := m.monitor.Start(m.ctx)
	if err != nil {
		m.errCh <- fmt.Errorf("failed to start clipboard monitor: %w", err)
		return
	}

	var droppedCount int

	for {
		select {
		case content, ok := <-contentChan:
			if !ok {
				m.errCh <- nil
				return
			}

			if !m.limiter.Allow() {
				droppedCount++
				if droppedCount%10 == 1 {
					slog.Warn("Rate limit exceeded, dropping clipboard scan",
						"dropped_total", droppedCount)
				}
				content.Wipe()
				continue
			}

			if droppedCount > 0 {
				slog.Info("Rate limit recovered", "dropped_total", droppedCount)
				droppedCount = 0
			}

			m.processContent(content)
		case <-m.ctx.Done():
			m.errCh <- m.ctx.Err()
			return
		}
	}
}

func (m *Monitor) processContent(content clipboard.Content) {
	defer content.Wipe()

	slog.Debug("Clipboard changed",
		"size_bytes", len(content.Text),
		"truncated", content.Truncated,
		"timestamp", content.Timestamp)

	findings := m.detector.Scan(content.Text)

	if len(findings) > 0 {
		slog.Info("Detected secrets in clipboard",
			"findings_count", len(findings))
		for _, finding := range findings {
			m.handleFinding(finding)
		}
	}
}

func (m *Monitor) handleFinding(finding detector.Finding) {
	slog.Info("Secret detected in clipboard",
		"pattern_id", finding.PatternID,
		"pattern_name", finding.PatternName,
		"description", finding.Description,
		"entropy", finding.Entropy,
		"match_length", len(finding.Match),
		"match", finding.Match,
		"start_index", finding.StartIndex,
		"end_index", finding.EndIndex)

	message := FormatSecretDetectedMessage(finding.PatternName)
	if err := m.notifier.Notify(NotificationTitleSecurityAlert, message); err != nil {
		slog.Error("Failed to send notification for detected secret",
			"error", err,
			"pattern_name", finding.PatternName)
	}
}

func FormatSecretDetectedMessage(patternName string) string {
	return fmt.Sprintf("Detected %s in clipboard", patternName)
}
