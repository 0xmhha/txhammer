package types

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// BlockResult holds statistics for a single block
type BlockResult struct {
	Number       uint64    `json:"number"`
	Time         time.Time `json:"time"`
	Transactions int       `json:"transactions"`
	GasUsed      uint64    `json:"gas_used"`
	GasLimit     uint64    `json:"gas_limit"`
}

// GasUtilization returns the gas utilization percentage
func (b *BlockResult) GasUtilization() float64 {
	if b.GasLimit == 0 {
		return 0
	}
	return float64(b.GasUsed) / float64(b.GasLimit) * 100
}

// RunResult holds the complete stress test results
type RunResult struct {
	// Summary
	TotalTransactions int     `json:"total_transactions"`
	SuccessfulTxs     int     `json:"successful_transactions"`
	FailedTxs         int     `json:"failed_transactions"`
	AverageTPS        float64 `json:"average_tps"`

	// Timing
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`

	// Block statistics
	StartBlock uint64         `json:"start_block"`
	EndBlock   uint64         `json:"end_block"`
	Blocks     []*BlockResult `json:"blocks"`

	// Gas statistics
	TotalGasUsed  uint64  `json:"total_gas_used"`
	AverageGasUtilization float64 `json:"average_gas_utilization"`

	// Transaction details
	TxHashes []common.Hash `json:"tx_hashes,omitempty"`
	FailedTxHashes []common.Hash `json:"failed_tx_hashes,omitempty"`
}

// TxBatchResult holds the result of a batch transaction send
type TxBatchResult struct {
	TxHashes   []common.Hash
	StartBlock uint64
	Errors     []error
}

// AccountInfo holds account information
type AccountInfo struct {
	Address common.Address
	Balance *big.Int
	Nonce   uint64
}

// GasSettings holds gas configuration
type GasSettings struct {
	GasLimit  uint64
	GasPrice  *big.Int
	GasTipCap *big.Int
	GasFeeCap *big.Int
}

// TransactionRequest represents a transaction to be built and sent
type TransactionRequest struct {
	From     common.Address
	To       *common.Address
	Value    *big.Int
	Data     []byte
	Nonce    uint64
	Gas      GasSettings
}

// FeeDelegationRequest extends TransactionRequest for fee delegation
type FeeDelegationRequest struct {
	TransactionRequest
	FeePayer common.Address
}
