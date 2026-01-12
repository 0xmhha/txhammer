package batcher

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/piatoss3612/txhammer/internal/txbuilder"
)

// TxStatus represents the status of a transaction
type TxStatus int

const (
	TxStatusPending TxStatus = iota
	TxStatusSent
	TxStatusConfirmed
	TxStatusFailed
)

func (s TxStatus) String() string {
	switch s {
	case TxStatusPending:
		return "PENDING"
	case TxStatusSent:
		return "SENT"
	case TxStatusConfirmed:
		return "CONFIRMED"
	case TxStatusFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// TxResult represents the result of a single transaction
type TxResult struct {
	Tx       *txbuilder.SignedTx
	Hash     common.Hash
	Status   TxStatus
	Error    error
	SentAt   time.Time
	BatchIdx int
}

// BatchResult represents the result of a batch send operation
type BatchResult struct {
	BatchIndex   int
	TxCount      int
	SuccessCount int
	FailedCount  int
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Results      []*TxResult
	Error        error
}

// Summary represents the overall batch operation summary
type Summary struct {
	TotalBatches   int
	TotalTxs       int
	SuccessCount   int
	FailedCount    int
	TotalDuration  time.Duration
	AvgBatchTime   time.Duration
	TxPerSecond    float64
	BatchResults   []*BatchResult
	FailedTxs      []*TxResult
}

// Config holds batcher configuration
type Config struct {
	// BatchSize is the number of transactions per batch
	BatchSize int

	// MaxConcurrent is the max concurrent batch requests
	MaxConcurrent int

	// BatchInterval is the delay between batches
	BatchInterval time.Duration

	// RetryCount is the number of retries for failed transactions
	RetryCount int

	// RetryDelay is the delay between retries
	RetryDelay time.Duration

	// Timeout is the timeout for batch operations
	Timeout time.Duration
}

// DefaultConfig returns default batcher configuration
func DefaultConfig() *Config {
	return &Config{
		BatchSize:     100,
		MaxConcurrent: 5,
		BatchInterval: 100 * time.Millisecond,
		RetryCount:    3,
		RetryDelay:    500 * time.Millisecond,
		Timeout:       30 * time.Second,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	if c.MaxConcurrent <= 0 {
		c.MaxConcurrent = 5
	}
	if c.BatchInterval < 0 {
		c.BatchInterval = 100 * time.Millisecond
	}
	if c.RetryCount < 0 {
		c.RetryCount = 0
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	return nil
}
