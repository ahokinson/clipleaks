package process

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
	"github.com/ahokinson/clipleaks/internal/notification"
	"github.com/ahokinson/clipleaks/internal/process/limit"
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

type Process struct {
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

func New(cfg *config.Config, stdout io.Writer) *Process {
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

	return &Process{
		cfg:      cfg,
		detector: det,
		notifier: notifier,
		monitor:  mon,
		stdout:   stdout,
		limiter:  limit.New(float64(*cfg.DetectionBurstLimit), *cfg.DetectionRateLimit),
		errCh:    make(chan error, ErrorChannelBuffer),
	}
}

func (p *Process) Run() error {
	return p.RunWithContext(context.Background())
}

func (p *Process) RunWithContext(ctx context.Context) error {
	p.initOnce.Do(func() {
		p.initErr = p.initialize()
	})
	if p.initErr != nil {
		return fmt.Errorf("failed to initialize process: %w", p.initErr)
	}

	p.ctx, p.cancel = context.WithCancel(ctx)
	defer func() { _ = p.Close() }()

	p.wg.Add(1)
	go p.handleSignals()

	p.wg.Add(1)
	go p.runMonitoring()

	if err := p.notifier.Notify(NotificationTitleStartup, NotificationMessageStartup); err != nil {
		slog.Error("Notification system is unavailable - this is required for security alerts", "error", err)
		p.cancel()
		p.wg.Wait()
		return fmt.Errorf("notification system unavailable: %w", err)
	}

	slog.Info("Clipleaks is now monitoring your clipboard - Press Ctrl+C to stop")

	err := <-p.errCh

	p.cancel()

	p.wg.Wait()

	select {
	case additionalErr := <-p.errCh:
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

func (p *Process) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func (p *Process) initialize() error {
	logger.Setup(p.stdout, slog.LevelInfo, false)
	return nil
}

func (p *Process) handleSignals() {
	defer p.wg.Done()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case sig := <-sigChan:
		slog.Info("Received shutdown signal", "signal", sig)
		p.errCh <- context.Canceled
	case <-p.ctx.Done():
		p.errCh <- p.ctx.Err()
	}
}

func (p *Process) runMonitoring() {
	defer p.wg.Done()

	contentChan, err := p.monitor.Start(p.ctx)
	if err != nil {
		p.errCh <- fmt.Errorf("failed to start clipboard monitor: %w", err)
		return
	}

	var droppedCount int

	for {
		select {
		case content, ok := <-contentChan:
			if !ok {
				p.errCh <- nil
				return
			}

			if !p.limiter.Allow() {
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

			p.processContent(content)
		case <-p.ctx.Done():
			p.errCh <- p.ctx.Err()
			return
		}
	}
}

func (p *Process) processContent(content clipboard.Content) {
	defer content.Wipe()

	slog.Debug("Clipboard changed",
		"size_bytes", len(content.Text),
		"truncated", content.Truncated,
		"timestamp", content.Timestamp)

	findings := p.detector.Scan(content.Text)

	if len(findings) > 0 {
		slog.Info("Detected secrets in clipboard",
			"findings_count", len(findings))
		for _, finding := range findings {
			p.handleFinding(finding)
		}
	}
}

func (p *Process) handleFinding(finding detector.Finding) {
	slog.Info("Secret detected in clipboard",
		"pattern_id", finding.PatternID,
		"pattern_name", finding.PatternName,
		"description", finding.Description,
		"entropy", finding.Entropy,
		"match_length", len(finding.Match),
		"start_index", finding.StartIndex,
		"end_index", finding.EndIndex)

	message := FormatSecretDetectedMessage(finding.PatternName)
	if err := p.notifier.Notify(NotificationTitleSecurityAlert, message); err != nil {
		slog.Error("Failed to send notification for detected secret",
			"error", err,
			"pattern_name", finding.PatternName)
	}
}

func FormatSecretDetectedMessage(patternName string) string {
	return fmt.Sprintf("Detected %s in clipboard", patternName)
}
