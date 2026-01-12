package distributor

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// AccountStatus represents the funding status of an account
type AccountStatus struct {
	Address      common.Address
	Balance      *big.Int
	RequiredFund *big.Int
	MissingFund  *big.Int
	Nonce        uint64
	IsFunded     bool
}

// DistributionResult holds the result of fund distribution
type DistributionResult struct {
	// Accounts that are ready for stress test
	ReadyAccounts []*AccountStatus

	// Accounts that could not be funded
	UnfundedAccounts []*AccountStatus

	// Total amount distributed
	TotalDistributed *big.Int

	// Number of distribution transactions sent
	TxCount int
}

// Config holds distribution configuration
type Config struct {
	// Amount of gas each sub-account needs per transaction
	GasPerTx uint64

	// Number of transactions each sub-account will send
	TxsPerAccount int

	// Gas price for calculations
	GasPrice *big.Int

	// Extra buffer percentage (e.g., 10 for 10% extra)
	BufferPercent int
}

// DefaultConfig returns default distribution configuration
func DefaultConfig() *Config {
	return &Config{
		GasPerTx:      21000,
		TxsPerAccount: 10,
		GasPrice:      big.NewInt(1000000000), // 1 Gwei
		BufferPercent: 20,                      // 20% buffer
	}
}

// CalculateRequiredFund calculates the required fund for an account
func (c *Config) CalculateRequiredFund() *big.Int {
	// Required = gasPerTx * txsPerAccount * gasPrice * (1 + buffer/100)
	baseCost := new(big.Int).Mul(
		big.NewInt(int64(c.GasPerTx)),
		big.NewInt(int64(c.TxsPerAccount)),
	)
	baseCost.Mul(baseCost, c.GasPrice)

	// Add buffer
	if c.BufferPercent > 0 {
		buffer := new(big.Int).Mul(baseCost, big.NewInt(int64(c.BufferPercent)))
		buffer.Div(buffer, big.NewInt(100))
		baseCost.Add(baseCost, buffer)
	}

	return baseCost
}
