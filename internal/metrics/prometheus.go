package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for txhammer
type Metrics struct {
	// Transaction counters
	TxSent      prometheus.Counter
	TxConfirmed prometheus.Counter
	TxFailed    prometheus.Counter
	TxTimeout   prometheus.Counter

	// Latency histogram (buckets: 100ms, 500ms, 1s, 2s, 5s, 10s, 30s, 60s)
	TxLatency prometheus.Histogram

	// Gauges for current state
	CurrentTPS     prometheus.Gauge
	ConfirmedTPS   prometheus.Gauge
	PendingTxCount prometheus.Gauge
	SendRate       prometheus.Gauge

	// Gas metrics
	GasUsedTotal prometheus.Counter

	// Pipeline stage duration histogram
	StageDuration *prometheus.HistogramVec

	// HTTP server
	server *http.Server
	mu     sync.Mutex
}

// NewMetrics creates a new Metrics instance with the given namespace
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		TxSent: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tx_sent_total",
			Help:      "Total number of transactions sent",
		}),
		TxConfirmed: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tx_confirmed_total",
			Help:      "Total number of transactions confirmed",
		}),
		TxFailed: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tx_failed_total",
			Help:      "Total number of transactions failed",
		}),
		TxTimeout: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tx_timeout_total",
			Help:      "Total number of transactions timed out",
		}),
		TxLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "tx_latency_seconds",
			Help:      "Transaction confirmation latency in seconds",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		}),
		CurrentTPS: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "current_tps",
			Help:      "Current transactions per second (send rate)",
		}),
		ConfirmedTPS: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "confirmed_tps",
			Help:      "Confirmed transactions per second",
		}),
		PendingTxCount: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "pending_tx_count",
			Help:      "Number of pending (unconfirmed) transactions",
		}),
		SendRate: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "send_rate",
			Help:      "Current send rate in transactions per second",
		}),
		GasUsedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "gas_used_total",
			Help:      "Total gas used by confirmed transactions",
		}),
		StageDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "stage_duration_seconds",
			Help:      "Duration of each pipeline stage in seconds",
			Buckets:   []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120, 300},
		}, []string{"stage"}),
	}

	return m
}

// Start starts the HTTP server for Prometheus metrics
func (m *Metrics) Start(_ context.Context, port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.server != nil {
		return fmt.Errorf("metrics server already running")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	m.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Metrics server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server gracefully
func (m *Metrics) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.server == nil {
		return nil
	}

	err := m.server.Shutdown(ctx)
	m.server = nil
	return err
}

// RecordTxSent increments the sent transaction counter
func (m *Metrics) RecordTxSent() {
	m.TxSent.Inc()
}

// RecordTxSentN increments the sent transaction counter by n
func (m *Metrics) RecordTxSentN(n int) {
	m.TxSent.Add(float64(n))
}

// RecordTxConfirmed increments the confirmed counter and records latency
func (m *Metrics) RecordTxConfirmed(latency time.Duration) {
	m.TxConfirmed.Inc()
	m.TxLatency.Observe(latency.Seconds())
}

// RecordTxFailed increments the failed transaction counter
func (m *Metrics) RecordTxFailed() {
	m.TxFailed.Inc()
}

// RecordTxTimeout increments the timeout counter
func (m *Metrics) RecordTxTimeout() {
	m.TxTimeout.Inc()
}

// SetCurrentTPS sets the current TPS gauge
func (m *Metrics) SetCurrentTPS(tps float64) {
	m.CurrentTPS.Set(tps)
}

// SetConfirmedTPS sets the confirmed TPS gauge
func (m *Metrics) SetConfirmedTPS(tps float64) {
	m.ConfirmedTPS.Set(tps)
}

// SetPendingCount sets the pending transaction count gauge
func (m *Metrics) SetPendingCount(count int) {
	m.PendingTxCount.Set(float64(count))
}

// SetSendRate sets the send rate gauge
func (m *Metrics) SetSendRate(rate float64) {
	m.SendRate.Set(rate)
}

// RecordGasUsed adds to the total gas used counter
func (m *Metrics) RecordGasUsed(gasUsed uint64) {
	m.GasUsedTotal.Add(float64(gasUsed))
}

// RecordStageDuration records the duration of a pipeline stage
func (m *Metrics) RecordStageDuration(stage string, duration time.Duration) {
	m.StageDuration.WithLabelValues(stage).Observe(duration.Seconds())
}

// IsRunning returns true if the metrics server is running
func (m *Metrics) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.server != nil
}
