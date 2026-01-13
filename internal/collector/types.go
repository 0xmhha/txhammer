package collector

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TxConfirmStatus represents the confirmation status of a transaction
type TxConfirmStatus int

const (
	TxConfirmPending TxConfirmStatus = iota
	TxConfirmSuccess
	TxConfirmFailed
	TxConfirmTimeout
	TxConfirmNotFound
)

func (s TxConfirmStatus) String() string {
	switch s {
	case TxConfirmPending:
		return "PENDING"
	case TxConfirmSuccess:
		return "SUCCESS"
	case TxConfirmFailed:
		return "FAILED"
	case TxConfirmTimeout:
		return "TIMEOUT"
	case TxConfirmNotFound:
		return "NOT_FOUND"
	default:
		return "UNKNOWN"
	}
}

// TxInfo represents tracked transaction information
type TxInfo struct {
	Hash        common.Hash
	From        common.Address
	Nonce       uint64
	GasLimit    uint64
	SentAt      time.Time
	ConfirmedAt time.Time
	Status      TxConfirmStatus
	Receipt     *types.Receipt
	Latency     time.Duration
	Error       error
}

// BlockInfo represents block-level metrics
type BlockInfo struct {
	Number       uint64
	Hash         common.Hash
	Timestamp    time.Time
	GasLimit     uint64
	GasUsed      uint64
	TxCount      int
	OurTxCount   int
	BaseFee      *big.Int
	Utilization  float64
}

// Metrics represents collected performance metrics
type Metrics struct {
	// Transaction metrics
	TotalSent      int
	TotalConfirmed int
	TotalFailed    int
	TotalPending   int
	TotalTimeout   int

	// Timing metrics
	StartTime       time.Time
	EndTime         time.Time
	TotalDuration   time.Duration
	AvgLatency      time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	P50Latency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration

	// Throughput metrics
	TPS             float64
	ConfirmedTPS    float64
	PeakTPS         float64

	// Gas metrics
	TotalGasUsed    uint64
	AvgGasUsed      uint64
	TotalGasCost    *big.Int
	AvgGasCost      *big.Int

	// Block metrics
	BlocksObserved  int
	AvgBlockTime    time.Duration
	AvgTxPerBlock   float64
	AvgUtilization  float64

	// Block-based TPS (transactions per block span)
	FirstBlockWithTx uint64  // First block containing our transactions
	LastBlockWithTx  uint64  // Last block containing our transactions
	BlockSpan        int     // Number of blocks (last - first + 1)
	BlocksWithOurTx  int     // Count of blocks that contain our transactions
	BlockBasedTPS    float64 // TotalConfirmed / (BlocksWithOurTx × AvgBlockTime)

	// Success rate
	SuccessRate     float64
}

// Config holds collector configuration
type Config struct {
	// PollInterval is the interval for polling receipts
	PollInterval time.Duration

	// ConfirmTimeout is the timeout for waiting for confirmation
	ConfirmTimeout time.Duration

	// MaxConcurrent is the max concurrent receipt queries
	MaxConcurrent int

	// BatchSize is the number of receipts to query in batch
	BatchSize int

	// BlockTrackingEnabled enables block-level metric tracking
	BlockTrackingEnabled bool

	// BlockPollInterval is the interval for polling blocks
	BlockPollInterval time.Duration
}

// DefaultConfig returns default collector configuration
func DefaultConfig() *Config {
	return &Config{
		PollInterval:         500 * time.Millisecond,
		ConfirmTimeout:       60 * time.Second,
		MaxConcurrent:        20,
		BatchSize:            100,
		BlockTrackingEnabled: true,
		BlockPollInterval:    1 * time.Second,
	}
}

// Report represents the final collection report
type Report struct {
	// Summary
	TestName    string
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration

	// Metrics
	Metrics *Metrics

	// Detailed results
	Transactions []*TxInfo
	Blocks       []*BlockInfo

	// Latency distribution
	LatencyHistogram map[string]int

	// Error summary
	ErrorSummary map[string]int
}

// NewReport creates a new report
func NewReport(testName string) *Report {
	return &Report{
		TestName:         testName,
		StartTime:        time.Now(),
		Metrics:          &Metrics{StartTime: time.Now()},
		Transactions:     make([]*TxInfo, 0),
		Blocks:           make([]*BlockInfo, 0),
		LatencyHistogram: make(map[string]int),
		ErrorSummary:     make(map[string]int),
	}
}
