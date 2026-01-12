package pipeline

import (
	"time"

	"github.com/piatoss3612/txhammer/internal/collector"
)

// Stage represents a pipeline stage
type Stage int

const (
	StageInit Stage = iota
	StageDistribute
	StageBuild
	StageSend
	StageCollect
	StageReport
	StageComplete
)

func (s Stage) String() string {
	switch s {
	case StageInit:
		return "INITIALIZE"
	case StageDistribute:
		return "DISTRIBUTE"
	case StageBuild:
		return "BUILD"
	case StageSend:
		return "SEND"
	case StageCollect:
		return "COLLECT"
	case StageReport:
		return "REPORT"
	case StageComplete:
		return "COMPLETE"
	default:
		return "UNKNOWN"
	}
}

// StageResult represents the result of a pipeline stage
type StageResult struct {
	Stage     Stage
	Success   bool
	Duration  time.Duration
	Message   string
	Error     error
}

// RunConfig holds runtime configuration for the pipeline
type RunConfig struct {
	// Skip distribution if accounts already funded
	SkipDistribution bool

	// Skip collection (fire-and-forget mode)
	SkipCollection bool

	// Export report to files
	ExportReport bool

	// Output directory for reports
	OutputDir string

	// Use streaming mode instead of batch mode
	StreamingMode bool

	// Rate limit for streaming mode (tx/s)
	StreamingRate float64

	// Dry run (build transactions but don't send)
	DryRun bool
}

// DefaultRunConfig returns default run configuration
func DefaultRunConfig() *RunConfig {
	return &RunConfig{
		SkipDistribution: false,
		SkipCollection:   false,
		ExportReport:     true,
		OutputDir:        "./reports",
		StreamingMode:    false,
		StreamingRate:    1000,
		DryRun:           false,
	}
}

// Result represents the complete pipeline execution result
type Result struct {
	// Execution info
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration

	// Stage results
	StageResults []*StageResult

	// Summary
	TotalTransactions int
	SuccessfulTxs     int
	FailedTxs         int
	TimeoutTxs        int

	// Performance metrics
	TPS          float64
	ConfirmedTPS float64
	AvgLatency   time.Duration
	P95Latency   time.Duration
	P99Latency   time.Duration

	// Gas metrics
	TotalGasUsed uint64
	TotalGasCost string

	// Detailed report
	Report *collector.Report

	// Errors encountered
	Errors []error
}

// NewResult creates a new pipeline result
func NewResult() *Result {
	return &Result{
		StartTime:    time.Now(),
		StageResults: make([]*StageResult, 0),
		Errors:       make([]error, 0),
	}
}

// AddStageResult adds a stage result
func (r *Result) AddStageResult(sr *StageResult) {
	r.StageResults = append(r.StageResults, sr)
	if sr.Error != nil {
		r.Errors = append(r.Errors, sr.Error)
	}
}

// Finalize completes the result
func (r *Result) Finalize() {
	r.EndTime = time.Now()
	r.Duration = r.EndTime.Sub(r.StartTime)
}

// Success returns true if all stages succeeded
func (r *Result) Success() bool {
	for _, sr := range r.StageResults {
		if !sr.Success {
			return false
		}
	}
	return true
}
