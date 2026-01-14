package analyzer

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

// Client defines the interface for block analysis
type Client interface {
	// BlockNumber returns the latest block number
	BlockNumber(ctx context.Context) (uint64, error)
	// BlockByNumber returns a block by its number
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

// Config holds configuration for the analyzer
type Config struct {
	StartBlock  int64 // Start block number (0 = calculate from BlockRange)
	EndBlock    int64 // End block number (0 = latest)
	BlockRange  int64 // Number of recent blocks to analyze
	Concurrency int   // Number of concurrent block fetches
}

// DefaultConfig returns default analyzer configuration
func DefaultConfig() *Config {
	return &Config{
		StartBlock:  0,
		EndBlock:    0,
		BlockRange:  100,
		Concurrency: 50,
	}
}

// BlockInfo holds information about a single block
type BlockInfo struct {
	Number      uint64
	Timestamp   time.Time
	TxCount     int
	GasLimit    uint64
	GasUsed     uint64
	Utilization float64       // Gas utilization percentage
	BlockTime   time.Duration // Time since previous block
}

// AnalysisResult holds the complete analysis results
type AnalysisResult struct {
	StartBlock    uint64
	EndBlock      uint64
	Blocks        []BlockInfo
	TotalTxs      uint64
	TotalDuration time.Duration
	AverageTPS    float64
	AvgBlockTime  time.Duration
	AvgGasUsed    float64
	AvgTxPerBlock float64
	MaxTxPerBlock int
	MinTxPerBlock int
}
