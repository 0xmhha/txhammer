package txbuilder

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/schollz/progressbar/v3"

	"github.com/0xmhha/txhammer/internal/config"
	"github.com/0xmhha/txhammer/internal/util/progress"
)

// TransferBuilder builds simple native coin transfer transactions (EIP-1559)
type TransferBuilder struct {
	*BaseBuilder
	recipient common.Address // If zero, transfers to self
}

// NewTransferBuilder creates a new transfer builder
func NewTransferBuilder(config *BuilderConfig, estimator GasEstimator) *TransferBuilder {
	return &TransferBuilder{
		BaseBuilder: NewBaseBuilder(config, estimator),
	}
}

// WithRecipient sets the recipient address
func (b *TransferBuilder) WithRecipient(addr common.Address) *TransferBuilder {
	b.recipient = addr
	return b
}

// Name returns the builder name
func (b *TransferBuilder) Name() string {
	return string(config.ModeTransfer)
}

// EstimateGas estimates gas for a simple transfer
func (b *TransferBuilder) EstimateGas(_ context.Context) (uint64, error) {
	// Simple ETH transfer is always 21000 gas
	return 21000, nil
}

// Build creates transfer transactions for the given accounts
func (b *TransferBuilder) Build(ctx context.Context, keys []*ecdsa.PrivateKey, nonces []uint64, count int) ([]*SignedTx, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys provided")
	}
	if len(keys) != len(nonces) {
		return nil, fmt.Errorf("keys and nonces length mismatch: %d vs %d", len(keys), len(nonces))
	}

	// Get gas settings (only need gasFeeCap for legacy transactions)
	_, gasFeeCap, err := b.GetGasSettings(ctx)
	if err != nil {
		return nil, err
	}

	// Use default gas limit if not configured
	gasLimit := b.config.GasLimit
	if gasLimit == 0 {
		gasLimit = 21000
	}

	// Distribute transactions across accounts
	distribution := DistributeTransactions(len(keys), count)

	// Calculate total transactions
	totalTxs := 0
	for _, n := range distribution {
		totalTxs += n
	}

	fmt.Printf("\nBuilding Transfer Transactions\n\n")
	bar := progressbar.Default(int64(totalTxs), "txs built")

	signedTxs := make([]*SignedTx, 0, totalTxs)

	// Build transactions for each account
	for accountIdx, txCount := range distribution {
		key := keys[accountIdx]
		nonce := nonces[accountIdx]
		from := crypto.PubkeyToAddress(key.PublicKey)

		for i := 0; i < txCount; i++ {
			// Determine recipient (self-transfer if not specified)
			to := b.recipient
			if to == (common.Address{}) {
				to = from
			}

			// Determine transfer value (default: 1 wei)
			value := b.config.Value
			if value == nil {
				value = big.NewInt(1)
			}

			// Create legacy transaction (type 0) for better compatibility
			tx := types.NewTx(&types.LegacyTx{
				Nonce:    nonce,
				GasPrice: gasFeeCap, // Use gasFeeCap as legacy gas price
				Gas:      gasLimit,
				To:       &to,
				Value:    value,
				Data:     nil,
			})

			// Sign the transaction
			signedTx, err := SignTransaction(tx, b.config.ChainID, key)
			if err != nil {
				return nil, fmt.Errorf("failed to sign transaction: %w", err)
			}

			// Encode to raw bytes
			rawTx, err := signedTx.MarshalBinary()
			if err != nil {
				return nil, fmt.Errorf("failed to marshal transaction: %w", err)
			}

			signedTxs = append(signedTxs, &SignedTx{
				Tx:       signedTx,
				RawTx:    rawTx,
				Hash:     signedTx.Hash(),
				From:     from,
				Nonce:    nonce,
				GasLimit: gasLimit,
			})

			nonce++
			progress.Add(bar, 1)
		}
	}

	fmt.Printf("\n[OK] Successfully built %d transactions\n", len(signedTxs))
	return signedTxs, nil
}

// BuildSingle creates a single transfer transaction
func (b *TransferBuilder) BuildSingle(
	ctx context.Context,
	key *ecdsa.PrivateKey,
	nonce uint64,
	to common.Address,
	value *big.Int,
) (*SignedTx, error) {
	_, gasFeeCap, err := b.GetGasSettings(ctx)
	if err != nil {
		return nil, err
	}

	gasLimit := b.config.GasLimit
	if gasLimit == 0 {
		gasLimit = 21000
	}

	from := crypto.PubkeyToAddress(key.PublicKey)

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasFeeCap, // Use gasFeeCap as legacy gas price
		Gas:      gasLimit,
		To:       &to,
		Value:    value,
		Data:     nil,
	})

	signedTx, err := SignTransaction(tx, b.config.ChainID, key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	rawTx, err := signedTx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	return &SignedTx{
		Tx:       signedTx,
		RawTx:    rawTx,
		Hash:     signedTx.Hash(),
		From:     from,
		Nonce:    nonce,
		GasLimit: gasLimit,
	}, nil
}
