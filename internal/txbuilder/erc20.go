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

	"github.com/0xmhha/txhammer/internal/util/progress"
)

// ERC20 function selectors
var (
	// transfer(address,uint256) = 0xa9059cbb
	ERC20TransferSelector = common.FromHex("0xa9059cbb")
	// balanceOf(address) = 0x70a08231
	ERC20BalanceOfSelector = common.FromHex("0x70a08231")
	// approve(address,uint256) = 0x095ea7b3
	ERC20ApproveSelector = common.FromHex("0x095ea7b3")
)

// ERC20TransferBuilder builds ERC20 transfer transactions
type ERC20TransferBuilder struct {
	*BaseBuilder
	tokenAddr common.Address
	recipient common.Address
	amount    *big.Int
}

// NewERC20TransferBuilder creates a new ERC20 transfer builder
func NewERC20TransferBuilder(config *BuilderConfig, estimator GasEstimator, tokenAddr common.Address) *ERC20TransferBuilder {
	return &ERC20TransferBuilder{
		BaseBuilder: NewBaseBuilder(config, estimator),
		tokenAddr:   tokenAddr,
		amount:      big.NewInt(1), // Default 1 token unit
	}
}

// WithRecipient sets the recipient address
func (b *ERC20TransferBuilder) WithRecipient(addr common.Address) *ERC20TransferBuilder {
	b.recipient = addr
	return b
}

// WithAmount sets the transfer amount
func (b *ERC20TransferBuilder) WithAmount(amount *big.Int) *ERC20TransferBuilder {
	b.amount = amount
	return b
}

// Name returns the builder name
func (b *ERC20TransferBuilder) Name() string {
	return "ERC20_TRANSFER"
}

// EstimateGas estimates gas for ERC20 transfer
func (b *ERC20TransferBuilder) EstimateGas(_ context.Context) (uint64, error) {
	// ERC20 transfer typically costs around 65000 gas
	return 65000, nil
}

// Build creates ERC20 transfer transactions
func (b *ERC20TransferBuilder) Build(ctx context.Context, keys []*ecdsa.PrivateKey, nonces []uint64, count int) ([]*SignedTx, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys provided")
	}
	if len(keys) != len(nonces) {
		return nil, fmt.Errorf("keys and nonces length mismatch")
	}
	if b.tokenAddr == (common.Address{}) {
		return nil, fmt.Errorf("token address is required")
	}

	gasTipCap, gasFeeCap, err := b.GetGasSettings(ctx)
	if err != nil {
		return nil, err
	}

	gasLimit := b.config.GasLimit
	if gasLimit == 0 {
		gasLimit = 65000
	}

	distribution := DistributeTransactions(len(keys), count)

	totalTxs := 0
	for _, n := range distribution {
		totalTxs += n
	}

	fmt.Printf("\nBuilding ERC20 Transfer Transactions\n\n")
	fmt.Printf("Token: %s\n", b.tokenAddr.Hex())
	bar := progressbar.Default(int64(totalTxs), "txs built")

	signedTxs := make([]*SignedTx, 0, totalTxs)

	for accountIdx, txCount := range distribution {
		key := keys[accountIdx]
		nonce := nonces[accountIdx]
		from := crypto.PubkeyToAddress(key.PublicKey)

		for i := 0; i < txCount; i++ {
			// Determine recipient (self-transfer if not specified)
			recipient := b.recipient
			if recipient == (common.Address{}) {
				recipient = from
			}

			// Build ERC20 transfer data
			data := buildERC20TransferData(recipient, b.amount)

			tx := types.NewTx(&types.DynamicFeeTx{
				ChainID:   b.config.ChainID,
				Nonce:     nonce,
				GasTipCap: gasTipCap,
				GasFeeCap: gasFeeCap,
				Gas:       gasLimit,
				To:        &b.tokenAddr,
				Value:     big.NewInt(0), // No native value for ERC20 transfer
				Data:      data,
			})

			signedTx, err := SignTransaction(tx, b.config.ChainID, key)
			if err != nil {
				return nil, fmt.Errorf("failed to sign transaction: %w", err)
			}

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

	fmt.Printf("\n[OK] Successfully built %d ERC20 transfer transactions\n", len(signedTxs))
	return signedTxs, nil
}

// buildERC20TransferData builds the calldata for ERC20 transfer(address,uint256)
func buildERC20TransferData(to common.Address, amount *big.Int) []byte {
	// transfer(address,uint256) selector = 0xa9059cbb
	data := make([]byte, 4+32+32)
	copy(data[0:4], ERC20TransferSelector)

	// address parameter (padded to 32 bytes)
	copy(data[4+12:4+32], to.Bytes())

	// uint256 parameter (padded to 32 bytes)
	amountBytes := amount.Bytes()
	copy(data[4+32+(32-len(amountBytes)):4+64], amountBytes)

	return data
}
