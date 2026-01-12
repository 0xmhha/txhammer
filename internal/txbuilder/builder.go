package txbuilder

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Builder interface defines the contract for transaction builders
type Builder interface {
	// Build creates transactions for the given accounts
	Build(ctx context.Context, keys []*ecdsa.PrivateKey, nonces []uint64, count int) ([]*SignedTx, error)
	// EstimateGas estimates gas for a single transaction
	EstimateGas(ctx context.Context) (uint64, error)
	// Name returns the builder name
	Name() string
}

// GasEstimator interface for gas estimation
type GasEstimator interface {
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
}

// BaseBuilder provides common functionality for all builders
type BaseBuilder struct {
	config    *BuilderConfig
	estimator GasEstimator
}

// NewBaseBuilder creates a new base builder
func NewBaseBuilder(config *BuilderConfig, estimator GasEstimator) *BaseBuilder {
	return &BaseBuilder{
		config:    config,
		estimator: estimator,
	}
}

// GetGasSettings returns gas settings, fetching from network if not configured
func (b *BaseBuilder) GetGasSettings(ctx context.Context) (*big.Int, *big.Int, error) {
	gasTipCap := b.config.GasTipCap
	gasFeeCap := b.config.GasFeeCap

	if gasTipCap == nil && b.estimator != nil {
		tip, err := b.estimator.SuggestGasTipCap(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to suggest gas tip cap: %w", err)
		}
		gasTipCap = tip
	}

	if gasFeeCap == nil && b.estimator != nil {
		price, err := b.estimator.SuggestGasPrice(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to suggest gas price: %w", err)
		}
		// gasFeeCap = baseFee + gasTipCap (approximate with 2x suggested price)
		gasFeeCap = new(big.Int).Mul(price, big.NewInt(2))
	}

	// Ensure gasTipCap is not greater than gasFeeCap
	if gasTipCap != nil && gasFeeCap != nil && gasTipCap.Cmp(gasFeeCap) > 0 {
		gasTipCap = gasFeeCap
	}

	return gasTipCap, gasFeeCap, nil
}

// SignTransaction signs a transaction with the given private key
func SignTransaction(tx *types.Transaction, chainID *big.Int, key *ecdsa.PrivateKey) (*types.Transaction, error) {
	signer := types.NewLondonSigner(chainID)
	return types.SignTx(tx, signer, key)
}

// AddressFromKey returns the address for a private key
func AddressFromKey(key *ecdsa.PrivateKey) common.Address {
	return crypto.PubkeyToAddress(key.PublicKey)
}

// DistributeTransactions distributes transactions across multiple accounts
// Returns a map of account index to number of transactions for that account
func DistributeTransactions(numAccounts, totalTxs int) map[int]int {
	distribution := make(map[int]int)
	if numAccounts == 0 {
		return distribution
	}

	baseCount := totalTxs / numAccounts
	remainder := totalTxs % numAccounts

	for i := 0; i < numAccounts; i++ {
		count := baseCount
		if i < remainder {
			count++
		}
		if count > 0 {
			distribution[i] = count
		}
	}

	return distribution
}
