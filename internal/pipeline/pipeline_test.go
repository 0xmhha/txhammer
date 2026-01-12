package pipeline

import (
	"testing"
	"time"
)

func TestStage_String(t *testing.T) {
	tests := []struct {
		stage    Stage
		expected string
	}{
		{StageInit, "INITIALIZE"},
		{StageDistribute, "DISTRIBUTE"},
		{StageBuild, "BUILD"},
		{StageSend, "SEND"},
		{StageCollect, "COLLECT"},
		{StageReport, "REPORT"},
		{StageComplete, "COMPLETE"},
		{Stage(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.stage.String(); got != tt.expected {
				t.Errorf("Stage.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDefaultRunConfig(t *testing.T) {
	cfg := DefaultRunConfig()

	if cfg == nil {
		t.Fatal("DefaultRunConfig() returned nil")
	}

	// Verify defaults
	if cfg.SkipDistribution {
		t.Error("SkipDistribution should be false by default")
	}
	if cfg.SkipCollection {
		t.Error("SkipCollection should be false by default")
	}
	if !cfg.ExportReport {
		t.Error("ExportReport should be true by default")
	}
	if cfg.OutputDir != "./reports" {
		t.Errorf("OutputDir = %v, want ./reports", cfg.OutputDir)
	}
	if cfg.StreamingMode {
		t.Error("StreamingMode should be false by default")
	}
	if cfg.StreamingRate != 1000 {
		t.Errorf("StreamingRate = %v, want 1000", cfg.StreamingRate)
	}
	if cfg.DryRun {
		t.Error("DryRun should be false by default")
	}
}

func TestNewResult(t *testing.T) {
	result := NewResult()

	if result == nil {
		t.Fatal("NewResult() returned nil")
	}

	if result.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
	if result.StageResults == nil {
		t.Error("StageResults should be initialized")
	}
	if result.Errors == nil {
		t.Error("Errors should be initialized")
	}
}

func TestResult_AddStageResult(t *testing.T) {
	result := NewResult()

	// Add successful stage
	sr1 := &StageResult{
		Stage:    StageInit,
		Success:  true,
		Duration: 1 * time.Second,
		Message:  "Completed",
	}
	result.AddStageResult(sr1)

	if len(result.StageResults) != 1 {
		t.Errorf("expected 1 stage result, got %d", len(result.StageResults))
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(result.Errors))
	}

	// Add failed stage
	sr2 := &StageResult{
		Stage:    StageBuild,
		Success:  false,
		Duration: 500 * time.Millisecond,
		Message:  "Failed",
		Error:    errTestError,
	}
	result.AddStageResult(sr2)

	if len(result.StageResults) != 2 {
		t.Errorf("expected 2 stage results, got %d", len(result.StageResults))
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestResult_Success(t *testing.T) {
	tests := []struct {
		name     string
		stages   []*StageResult
		expected bool
	}{
		{
			name:     "no stages",
			stages:   []*StageResult{},
			expected: true,
		},
		{
			name: "all success",
			stages: []*StageResult{
				{Stage: StageInit, Success: true},
				{Stage: StageBuild, Success: true},
				{Stage: StageSend, Success: true},
			},
			expected: true,
		},
		{
			name: "one failure",
			stages: []*StageResult{
				{Stage: StageInit, Success: true},
				{Stage: StageBuild, Success: false},
				{Stage: StageSend, Success: true},
			},
			expected: false,
		},
		{
			name: "all failures",
			stages: []*StageResult{
				{Stage: StageInit, Success: false},
				{Stage: StageBuild, Success: false},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewResult()
			for _, sr := range tt.stages {
				result.AddStageResult(sr)
			}
			if got := result.Success(); got != tt.expected {
				t.Errorf("Result.Success() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResult_Finalize(t *testing.T) {
	result := NewResult()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure duration > 0

	result.Finalize()

	if result.EndTime.IsZero() {
		t.Error("EndTime should be set after Finalize()")
	}
	if result.Duration <= 0 {
		t.Error("Duration should be positive after Finalize()")
	}
	if !result.EndTime.After(result.StartTime) {
		t.Error("EndTime should be after StartTime")
	}
}

func TestRunConfig_Customization(t *testing.T) {
	cfg := &RunConfig{
		SkipDistribution: true,
		SkipCollection:   true,
		ExportReport:     false,
		OutputDir:        "/custom/path",
		StreamingMode:    true,
		StreamingRate:    500,
		DryRun:           true,
	}

	if !cfg.SkipDistribution {
		t.Error("SkipDistribution should be true")
	}
	if !cfg.SkipCollection {
		t.Error("SkipCollection should be true")
	}
	if cfg.ExportReport {
		t.Error("ExportReport should be false")
	}
	if cfg.OutputDir != "/custom/path" {
		t.Errorf("OutputDir = %v, want /custom/path", cfg.OutputDir)
	}
	if !cfg.StreamingMode {
		t.Error("StreamingMode should be true")
	}
	if cfg.StreamingRate != 500 {
		t.Errorf("StreamingRate = %v, want 500", cfg.StreamingRate)
	}
	if !cfg.DryRun {
		t.Error("DryRun should be true")
	}
}

func TestStageResult_Fields(t *testing.T) {
	sr := &StageResult{
		Stage:    StageSend,
		Success:  true,
		Duration: 2 * time.Second,
		Message:  "Sent 100 transactions",
		Error:    nil,
	}

	if sr.Stage != StageSend {
		t.Errorf("Stage = %v, want StageSend", sr.Stage)
	}
	if !sr.Success {
		t.Error("Success should be true")
	}
	if sr.Duration != 2*time.Second {
		t.Errorf("Duration = %v, want 2s", sr.Duration)
	}
	if sr.Message != "Sent 100 transactions" {
		t.Errorf("Message = %v, want 'Sent 100 transactions'", sr.Message)
	}
	if sr.Error != nil {
		t.Error("Error should be nil")
	}
}

// Test error type for testing
type testError struct{}

func (e testError) Error() string { return "test error" }

var errTestError = testError{}
