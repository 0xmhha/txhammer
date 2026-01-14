package longsender

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// SendClient defines the interface for sending transactions
type SendClient interface {
	// SendTransaction sends a signed transaction to the network
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	// PendingNonceAt returns the pending nonce for an account
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	// SuggestGasPrice suggests a gas price
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	// ChainID returns the chain ID
	ChainID(ctx context.Context) (*big.Int, error)
}

// Config holds configuration for the LongSender
type Config struct {
	Duration time.Duration // Total test duration (0 = run until canceled)
	TPS      float64       // Target transactions per second
	Burst    int           // Rate limiter burst size
	Workers  int           // Number of concurrent workers
}

// DefaultConfig returns default LongSender configuration
func DefaultConfig() *Config {
	return &Config{
		Duration: 0,   // Run indefinitely
		TPS:      100, // 100 tx/s
		Burst:    10,  // Burst of 10
		Workers:  10,  // 10 workers
	}
}

// Result holds the results of a long sender run
type Result struct {
	TotalSent     int64
	TotalFailed   int64
	TotalDuration time.Duration
	AverageTPS    float64
	ActualTPS     float64
	Errors        []error
}

// Callbacks for metrics integration
type Callbacks struct {
	OnSent    func(hash common.Hash)
	OnFailed  func(err error)
	OnTPS     func(currentTPS float64)
	OnMetrics func(sent, failed int64, tps float64)
}
