package txbuilder

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/schollz/progressbar/v3"
)

const (
	// FeeDelegationTxType is the transaction type for fee delegation (StableNet specific)
	FeeDelegationTxType = 0x16
)

// FeeDelegationBuilder builds fee delegation transactions (Type 0x16)
// This is a StableNet-specific transaction type where a fee payer pays gas on behalf of the sender
type FeeDelegationBuilder struct {
	*BaseBuilder
	feePayerKey *ecdsa.PrivateKey
	recipient   common.Address
}

// NewFeeDelegationBuilder creates a new fee delegation builder
func NewFeeDelegationBuilder(config *BuilderConfig, estimator GasEstimator, feePayerKey *ecdsa.PrivateKey) *FeeDelegationBuilder {
	return &FeeDelegationBuilder{
		BaseBuilder: NewBaseBuilder(config, estimator),
		feePayerKey: feePayerKey,
	}
}

// WithRecipient sets the recipient address
func (b *FeeDelegationBuilder) WithRecipient(addr common.Address) *FeeDelegationBuilder {
	b.recipient = addr
	return b
}

// Name returns the builder name
func (b *FeeDelegationBuilder) Name() string {
	return "FEE_DELEGATION"
}

// EstimateGas estimates gas for a fee delegation transfer
func (b *FeeDelegationBuilder) EstimateGas(ctx context.Context) (uint64, error) {
	// Fee delegation adds some overhead, but base transfer is still 21000
	return 21000, nil
}

// Build creates fee delegation transactions for the given accounts
func (b *FeeDelegationBuilder) Build(ctx context.Context, keys []*ecdsa.PrivateKey, nonces []uint64, count int) ([]*SignedTx, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys provided")
	}
	if len(keys) != len(nonces) {
		return nil, fmt.Errorf("keys and nonces length mismatch: %d vs %d", len(keys), len(nonces))
	}
	if b.feePayerKey == nil {
		return nil, fmt.Errorf("fee payer key is required for fee delegation")
	}

	// Get gas settings
	gasTipCap, gasFeeCap, err := b.GetGasSettings(ctx)
	if err != nil {
		return nil, err
	}

	gasLimit := b.config.GasLimit
	if gasLimit == 0 {
		gasLimit = 21000
	}

	// Distribute transactions across accounts
	distribution := DistributeTransactions(len(keys), count)

	totalTxs := 0
	for _, n := range distribution {
		totalTxs += n
	}

	fmt.Printf("\n📝 Building Fee Delegation Transactions 📝\n\n")
	fmt.Printf("Fee Payer: %s\n", crypto.PubkeyToAddress(b.feePayerKey.PublicKey).Hex())
	bar := progressbar.Default(int64(totalTxs), "txs built")

	signedTxs := make([]*SignedTx, 0, totalTxs)
	feePayer := crypto.PubkeyToAddress(b.feePayerKey.PublicKey)

	for accountIdx, txCount := range distribution {
		key := keys[accountIdx]
		nonce := nonces[accountIdx]
		from := crypto.PubkeyToAddress(key.PublicKey)

		for i := 0; i < txCount; i++ {
			to := b.recipient
			if to == (common.Address{}) {
				to = from
			}

			// Build and sign fee delegation transaction
			rawTx, txHash, err := b.buildFeeDelegationTx(
				key,
				b.feePayerKey,
				nonce,
				to,
				big.NewInt(1), // 1 wei
				gasLimit,
				gasTipCap,
				gasFeeCap,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to build fee delegation tx: %w", err)
			}

			signedTxs = append(signedTxs, &SignedTx{
				Tx:       nil, // Fee delegation tx is not standard types.Transaction
				RawTx:    rawTx,
				Hash:     txHash,
				From:     from,
				Nonce:    nonce,
				GasLimit: gasLimit,
			})

			nonce++
			_ = bar.Add(1)
		}
	}

	fmt.Printf("\n✅ Successfully built %d fee delegation transactions\n", len(signedTxs))
	fmt.Printf("   Fee Payer: %s\n", feePayer.Hex())
	return signedTxs, nil
}

// buildFeeDelegationTx creates a single fee delegation transaction
// This follows the StableNet FeeDelegateDynamicFeeTx structure (Type 0x16)
func (b *FeeDelegationBuilder) buildFeeDelegationTx(
	senderKey *ecdsa.PrivateKey,
	feePayerKey *ecdsa.PrivateKey,
	nonce uint64,
	to common.Address,
	value *big.Int,
	gasLimit uint64,
	gasTipCap *big.Int,
	gasFeeCap *big.Int,
) ([]byte, common.Hash, error) {
	chainID := b.config.ChainID
	feePayer := crypto.PubkeyToAddress(feePayerKey.PublicKey)

	// Step 1: Create sender transaction hash (same as EIP-1559)
	// Hash = keccak256(0x02 || rlp([chainId, nonce, gasTipCap, gasFeeCap, gas, to, value, data, accessList]))
	senderTxData := []interface{}{
		chainID,
		nonce,
		gasTipCap,
		gasFeeCap,
		gasLimit,
		to,
		value,
		[]byte{},       // data
		types.AccessList{}, // accessList
	}

	senderHash := prefixedRlpHash(0x02, senderTxData)

	// Step 2: Sign sender transaction
	senderSig, err := crypto.Sign(senderHash[:], senderKey)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to sign sender tx: %w", err)
	}

	senderR := new(big.Int).SetBytes(senderSig[:32])
	senderS := new(big.Int).SetBytes(senderSig[32:64])
	senderV := new(big.Int).SetInt64(int64(senderSig[64]))

	// Step 3: Create fee payer hash
	// Hash = keccak256(0x16 || rlp([[senderTxData, senderV, senderR, senderS], feePayer]))
	senderTxWithSig := []interface{}{
		chainID,
		nonce,
		gasTipCap,
		gasFeeCap,
		gasLimit,
		to,
		value,
		[]byte{},
		types.AccessList{},
		senderV,
		senderR,
		senderS,
	}

	feePayerHashData := []interface{}{
		senderTxWithSig,
		feePayer,
	}

	feePayerHash := prefixedRlpHash(FeeDelegationTxType, feePayerHashData)

	// Step 4: Sign fee payer transaction
	feePayerSig, err := crypto.Sign(feePayerHash[:], feePayerKey)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to sign fee payer tx: %w", err)
	}

	feePayerR := new(big.Int).SetBytes(feePayerSig[:32])
	feePayerS := new(big.Int).SetBytes(feePayerSig[32:64])
	feePayerV := new(big.Int).SetInt64(int64(feePayerSig[64]))

	// Step 5: Create the full transaction
	// FeeDelegateDynamicFeeTx = 0x16 || rlp([SenderTx, FeePayer, FV, FR, FS])
	// where SenderTx = [chainId, nonce, gasTipCap, gasFeeCap, gas, to, value, data, accessList, V, R, S]
	fullTxData := []interface{}{
		senderTxWithSig,
		feePayer,
		feePayerV,
		feePayerR,
		feePayerS,
	}

	var buf bytes.Buffer
	buf.WriteByte(FeeDelegationTxType)
	if err := rlp.Encode(&buf, fullTxData); err != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to encode tx: %w", err)
	}

	rawTx := buf.Bytes()
	txHash := crypto.Keccak256Hash(rawTx)

	return rawTx, txHash, nil
}

// prefixedRlpHash computes keccak256(prefix || rlp(data))
func prefixedRlpHash(prefix byte, data interface{}) common.Hash {
	var buf bytes.Buffer
	buf.WriteByte(prefix)
	if err := rlp.Encode(&buf, data); err != nil {
		panic(fmt.Sprintf("failed to encode rlp: %v", err))
	}
	return crypto.Keccak256Hash(buf.Bytes())
}
