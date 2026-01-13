package monitor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Config holds configuration for the monitor
type Config struct {
	UpdateInterval time.Duration // How often to update display
	WindowSize     time.Duration // Rolling window for current TPS calculation
}

// DefaultConfig returns default monitor configuration
func DefaultConfig() *Config {
	return &Config{
		UpdateInterval: time.Second,
		WindowSize:     10 * time.Second,
	}
}

// sample represents a point-in-time measurement
type sample struct {
	timestamp time.Time
	sent      int64
	confirmed int64
}

// Monitor provides real-time TPS monitoring
type Monitor struct {
	config *Config

	// Atomic counters for thread-safe updates
	sentCount      atomic.Int64
	confirmedCount atomic.Int64
	failedCount    atomic.Int64

	// Timing
	startTime time.Time

	// Rolling window samples
	windowSamples []sample
	sampleMu      sync.Mutex

	// Last displayed values for delta calculation
	lastSent      int64
	lastConfirmed int64
	lastTime      time.Time
}

// Snapshot represents a point-in-time view of metrics
type Snapshot struct {
	TotalSent      int64
	TotalConfirmed int64
	TotalFailed    int64
	CurrentTPS     float64 // TPS in last window
	AvgTPS         float64 // TPS since start
	ConfirmedTPS   float64 // Confirmed TPS in last window
	Elapsed        time.Duration
}

// New creates a new Monitor instance
func New(config *Config) *Monitor {
	if config == nil {
		config = DefaultConfig()
	}
	return &Monitor{
		config:        config,
		windowSamples: make([]sample, 0, 100),
	}
}

// Start initializes the monitor with start time
func (m *Monitor) Start() {
	m.startTime = time.Now()
	m.lastTime = m.startTime
}

// RecordSent increments the sent counter by n
func (m *Monitor) RecordSent(n int64) {
	m.sentCount.Add(n)
	m.recordSample()
}

// RecordConfirmed increments the confirmed counter by n
func (m *Monitor) RecordConfirmed(n int64) {
	m.confirmedCount.Add(n)
}

// RecordFailed increments the failed counter by n
func (m *Monitor) RecordFailed(n int64) {
	m.failedCount.Add(n)
}

// recordSample adds a sample to the rolling window
func (m *Monitor) recordSample() {
	m.sampleMu.Lock()
	defer m.sampleMu.Unlock()

	now := time.Now()
	m.windowSamples = append(m.windowSamples, sample{
		timestamp: now,
		sent:      m.sentCount.Load(),
		confirmed: m.confirmedCount.Load(),
	})

	// Remove old samples outside the window
	cutoff := now.Add(-m.config.WindowSize)
	newStart := 0
	for i, s := range m.windowSamples {
		if s.timestamp.After(cutoff) {
			newStart = i
			break
		}
	}
	if newStart > 0 {
		m.windowSamples = m.windowSamples[newStart:]
	}
}

// Snapshot returns current metrics snapshot
func (m *Monitor) Snapshot() *Snapshot {
	now := time.Now()
	sent := m.sentCount.Load()
	confirmed := m.confirmedCount.Load()
	failed := m.failedCount.Load()
	elapsed := now.Sub(m.startTime)

	// Calculate average TPS since start
	avgTPS := float64(0)
	if elapsed.Seconds() > 0 {
		avgTPS = float64(sent) / elapsed.Seconds()
	}

	// Calculate current TPS from rolling window
	currentTPS := float64(0)
	confirmedTPS := float64(0)

	m.sampleMu.Lock()
	if len(m.windowSamples) >= 2 {
		first := m.windowSamples[0]
		last := m.windowSamples[len(m.windowSamples)-1]
		windowDuration := last.timestamp.Sub(first.timestamp).Seconds()
		if windowDuration > 0 {
			currentTPS = float64(last.sent-first.sent) / windowDuration
			confirmedTPS = float64(last.confirmed-first.confirmed) / windowDuration
		}
	}
	m.sampleMu.Unlock()

	return &Snapshot{
		TotalSent:      sent,
		TotalConfirmed: confirmed,
		TotalFailed:    failed,
		CurrentTPS:     currentTPS,
		AvgTPS:         avgTPS,
		ConfirmedTPS:   confirmedTPS,
		Elapsed:        elapsed,
	}
}

// DisplayLine returns a formatted single-line status
func (m *Monitor) DisplayLine() string {
	s := m.Snapshot()
	return fmt.Sprintf("Sent: %d | Confirmed: %d | Failed: %d | Current TPS: %.1f | Avg TPS: %.1f | Elapsed: %s",
		s.TotalSent, s.TotalConfirmed, s.TotalFailed, s.CurrentTPS, s.AvgTPS, formatDuration(s.Elapsed))
}

// Display starts a goroutine that periodically prints status
func (m *Monitor) Display(ctx context.Context) {
	ticker := time.NewTicker(m.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Printf("\r%s", m.DisplayLine())
		}
	}
}

// formatDuration formats duration as human readable string
func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// GetCurrentTPS returns the current TPS for external use
func (m *Monitor) GetCurrentTPS() float64 {
	s := m.Snapshot()
	return s.CurrentTPS
}

// GetAvgTPS returns the average TPS for external use
func (m *Monitor) GetAvgTPS() float64 {
	s := m.Snapshot()
	return s.AvgTPS
}
