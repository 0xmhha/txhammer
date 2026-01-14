package txbuilder

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xmhha/txhammer/internal/config"
)

const (
	testPrivateKey   = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	feePayerKeyHex   = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	testContractAddr = "0x1234567890123456789012345678901234567890"
	testTokenAddr    = "0xabcdef0123456789abcdef0123456789abcdef01"
)

// mockGasEstimator implements GasEstimator for testing
type mockGasEstimator struct {
	gasPrice  *big.Int
	gasTipCap *big.Int
	err       error
}

func (m *mockGasEstimator) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.gasPrice == nil {
		return big.NewInt(1000000000), nil // 1 Gwei
	}
	return m.gasPrice, nil
}

func (m *mockGasEstimator) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.gasTipCap == nil {
		return big.NewInt(100000000), nil // 0.1 Gwei
	}
	return m.gasTipCap, nil
}

func newTestKey() *ecdsa.PrivateKey {
	key, _ := crypto.HexToECDSA(testPrivateKey)
	return key
}

func newFeePayerKey() *ecdsa.PrivateKey {
	key, _ := crypto.HexToECDSA(feePayerKeyHex)
	return key
}

func TestDistributeTransactions(t *testing.T) {
	tests := []struct {
		name        string
		numAccounts int
		totalTxs    int
		wantTotal   int
	}{
		{
			name:        "equal distribution",
			numAccounts: 5,
			totalTxs:    100,
			wantTotal:   100,
		},
		{
			name:        "unequal distribution",
			numAccounts: 3,
			totalTxs:    100,
			wantTotal:   100,
		},
		{
			name:        "more accounts than txs",
			numAccounts: 10,
			totalTxs:    5,
			wantTotal:   5,
		},
		{
			name:        "single account",
			numAccounts: 1,
			totalTxs:    50,
			wantTotal:   50,
		},
		{
			name:        "zero accounts",
			numAccounts: 0,
			totalTxs:    100,
			wantTotal:   0,
		},
		{
			name:        "zero transactions",
			numAccounts: 5,
			totalTxs:    0,
			wantTotal:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := DistributeTransactions(tt.numAccounts, tt.totalTxs)

			// Calculate total
			total := 0
			for _, count := range dist {
				total += count
			}

			if total != tt.wantTotal {
				t.Errorf("DistributeTransactions() total = %d, want %d", total, tt.wantTotal)
			}

			// Verify distribution is balanced
			if tt.numAccounts > 0 && tt.totalTxs > 0 {
				var minCount, maxCount int
				first := true
				for _, count := range dist {
					if first {
						minCount, maxCount = count, count
						first = false
					} else {
						if count < minCount {
							minCount = count
						}
						if count > maxCount {
							maxCount = count
						}
					}
				}
				// Difference should be at most 1
				if maxCount-minCount > 1 {
					t.Errorf("Distribution not balanced: min=%d, max=%d", minCount, maxCount)
				}
			}
		})
	}
}

func TestAddressFromKey(t *testing.T) {
	key := newTestKey()
	addr := AddressFromKey(key)

	if addr == (common.Address{}) {
		t.Error("AddressFromKey() returned zero address")
	}

	// Verify it matches crypto.PubkeyToAddress
	expected := crypto.PubkeyToAddress(key.PublicKey)
	if addr != expected {
		t.Errorf("AddressFromKey() = %s, want %s", addr.Hex(), expected.Hex())
	}
}

func TestSignTransaction(t *testing.T) {
	key := newTestKey()
	chainID := big.NewInt(1001)

	// Create a simple transaction
	cfg := &BuilderConfig{
		ChainID:   chainID,
		GasLimit:  21000,
		GasTipCap: big.NewInt(100000000),
		GasFeeCap: big.NewInt(1000000000),
	}

	builder := NewTransferBuilder(cfg, nil)
	addr := crypto.PubkeyToAddress(key.PublicKey)

	signedTx, err := builder.BuildSingle(context.Background(), key, 0, addr, big.NewInt(1))
	if err != nil {
		t.Fatalf("BuildSingle() failed: %v", err)
	}

	if signedTx.Tx == nil {
		t.Error("SignedTx.Tx is nil")
	}

	if signedTx.Hash == (common.Hash{}) {
		t.Error("SignedTx.Hash is zero")
	}

	if len(signedTx.RawTx) == 0 {
		t.Error("SignedTx.RawTx is empty")
	}

	if signedTx.From != addr {
		t.Errorf("SignedTx.From = %s, want %s", signedTx.From.Hex(), addr.Hex())
	}

	if signedTx.Nonce != 0 {
		t.Errorf("SignedTx.Nonce = %d, want 0", signedTx.Nonce)
	}

	if signedTx.GasLimit != 21000 {
		t.Errorf("SignedTx.GasLimit = %d, want 21000", signedTx.GasLimit)
	}
}

func TestTransferBuilder_Name(t *testing.T) {
	cfg := &BuilderConfig{ChainID: big.NewInt(1)}
	builder := NewTransferBuilder(cfg, nil)

	if name := builder.Name(); name != "TRANSFER" {
		t.Errorf("Name() = %s, want TRANSFER", name)
	}
}

func TestTransferBuilder_EstimateGas(t *testing.T) {
	cfg := &BuilderConfig{ChainID: big.NewInt(1)}
	builder := NewTransferBuilder(cfg, nil)

	gas, err := builder.EstimateGas(context.Background())
	if err != nil {
		t.Fatalf("EstimateGas() error: %v", err)
	}

	if gas != 21000 {
		t.Errorf("EstimateGas() = %d, want 21000", gas)
	}
}

func TestTransferBuilder_Build_NoKeys(t *testing.T) {
	cfg := &BuilderConfig{
		ChainID:   big.NewInt(1),
		GasLimit:  21000,
		GasTipCap: big.NewInt(100000000),
		GasFeeCap: big.NewInt(1000000000),
	}
	builder := NewTransferBuilder(cfg, nil)

	_, err := builder.Build(context.Background(), nil, nil, 10)
	if err == nil {
		t.Error("Build() expected error for no keys")
	}
}

func TestTransferBuilder_Build_MismatchedLengths(t *testing.T) {
	cfg := &BuilderConfig{
		ChainID:   big.NewInt(1),
		GasLimit:  21000,
		GasTipCap: big.NewInt(100000000),
		GasFeeCap: big.NewInt(1000000000),
	}
	builder := NewTransferBuilder(cfg, nil)

	keys := []*ecdsa.PrivateKey{newTestKey()}
	nonces := []uint64{0, 1} // Mismatched length

	_, err := builder.Build(context.Background(), keys, nonces, 10)
	if err == nil {
		t.Error("Build() expected error for mismatched lengths")
	}
}

func TestTransferBuilder_Build_WithGasEstimator(t *testing.T) {
	cfg := &BuilderConfig{
		ChainID:  big.NewInt(1001),
		GasLimit: 21000,
	}

	estimator := &mockGasEstimator{
		gasPrice:  big.NewInt(2000000000),
		gasTipCap: big.NewInt(200000000),
	}

	builder := NewTransferBuilder(cfg, estimator)

	keys := []*ecdsa.PrivateKey{newTestKey()}
	nonces := []uint64{0}

	txs, err := builder.Build(context.Background(), keys, nonces, 5)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	if len(txs) != 5 {
		t.Errorf("Build() returned %d txs, want 5", len(txs))
	}

	// Verify all transactions are valid
	for i, tx := range txs {
		if tx.Hash == (common.Hash{}) {
			t.Errorf("tx[%d] has zero hash", i)
		}
		if len(tx.RawTx) == 0 {
			t.Errorf("tx[%d] has empty raw tx", i)
		}
		if tx.Nonce != uint64(i) {
			t.Errorf("tx[%d].Nonce = %d, want %d", i, tx.Nonce, i)
		}
	}
}

func TestFeeDelegationBuilder_Name(t *testing.T) {
	cfg := &BuilderConfig{ChainID: big.NewInt(1)}
	builder := NewFeeDelegationBuilder(cfg, nil, newFeePayerKey())

	if name := builder.Name(); name != "FEE_DELEGATION" {
		t.Errorf("Name() = %s, want FEE_DELEGATION", name)
	}
}

func TestFeeDelegationBuilder_EstimateGas(t *testing.T) {
	cfg := &BuilderConfig{ChainID: big.NewInt(1)}
	builder := NewFeeDelegationBuilder(cfg, nil, newFeePayerKey())

	gas, err := builder.EstimateGas(context.Background())
	if err != nil {
		t.Fatalf("EstimateGas() error: %v", err)
	}

	if gas != 21000 {
		t.Errorf("EstimateGas() = %d, want 21000", gas)
	}
}

func TestFeeDelegationBuilder_Build_NoFeePayerKey(t *testing.T) {
	cfg := &BuilderConfig{
		ChainID:   big.NewInt(1),
		GasLimit:  21000,
		GasTipCap: big.NewInt(100000000),
		GasFeeCap: big.NewInt(1000000000),
	}
	builder := NewFeeDelegationBuilder(cfg, nil, nil)

	keys := []*ecdsa.PrivateKey{newTestKey()}
	nonces := []uint64{0}

	_, err := builder.Build(context.Background(), keys, nonces, 10)
	if err == nil {
		t.Error("Build() expected error for no fee payer key")
	}
}

func TestFeeDelegationBuilder_Build(t *testing.T) {
	cfg := &BuilderConfig{
		ChainID:   big.NewInt(1001),
		GasLimit:  21000,
		GasTipCap: big.NewInt(100000000),
		GasFeeCap: big.NewInt(1000000000),
	}
	feePayerKey := newFeePayerKey()
	builder := NewFeeDelegationBuilder(cfg, nil, feePayerKey)

	keys := []*ecdsa.PrivateKey{newTestKey()}
	nonces := []uint64{0}

	txs, err := builder.Build(context.Background(), keys, nonces, 3)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	if len(txs) != 3 {
		t.Errorf("Build() returned %d txs, want 3", len(txs))
	}

	// Verify fee delegation transactions
	for i, tx := range txs {
		if tx.Hash == (common.Hash{}) {
			t.Errorf("tx[%d] has zero hash", i)
		}
		if len(tx.RawTx) == 0 {
			t.Errorf("tx[%d] has empty raw tx", i)
		}
		// Fee delegation tx has type prefix 0x16
		if tx.RawTx[0] != FeeDelegationTxType {
			t.Errorf("tx[%d] has wrong type prefix: %x, want %x", i, tx.RawTx[0], FeeDelegationTxType)
		}
		if tx.Nonce != uint64(i) {
			t.Errorf("tx[%d].Nonce = %d, want %d", i, tx.Nonce, i)
		}
		// Fee delegation tx has nil Tx field
		if tx.Tx != nil {
			t.Errorf("tx[%d].Tx should be nil for fee delegation", i)
		}
	}
}

func TestFactory_CreateBuilder_Transfer(t *testing.T) {
	cfg := &BuilderConfig{
		ChainID:  big.NewInt(1001),
		GasLimit: 21000,
	}
	factory := NewFactory(cfg, &mockGasEstimator{})

	builder, err := factory.CreateBuilder(config.ModeTransfer)
	if err != nil {
		t.Fatalf("CreateBuilder() error: %v", err)
	}

	if builder == nil {
		t.Fatal("CreateBuilder() returned nil")
	}

	if builder.Name() != "TRANSFER" {
		t.Errorf("Builder.Name() = %s, want TRANSFER", builder.Name())
	}
}

func TestFactory_CreateBuilder_FeeDelegation(t *testing.T) {
	cfg := &BuilderConfig{
		ChainID:  big.NewInt(1001),
		GasLimit: 21000,
	}
	factory := NewFactory(cfg, &mockGasEstimator{})

	// Without fee payer key - should fail
	_, err := factory.CreateBuilder(config.ModeFeeDelegation)
	if err == nil {
		t.Error("CreateBuilder() expected error without fee payer key")
	}

	// With fee payer key - should succeed
	builder, err := factory.CreateBuilder(config.ModeFeeDelegation, WithFeePayerKey(newFeePayerKey()))
	if err != nil {
		t.Fatalf("CreateBuilder() error: %v", err)
	}

	if builder.Name() != "FEE_DELEGATION" {
		t.Errorf("Builder.Name() = %s, want FEE_DELEGATION", builder.Name())
	}
}

func TestFactory_CreateBuilder_ContractCall_RequiresAddress(t *testing.T) {
	cfg := &BuilderConfig{
		ChainID:  big.NewInt(1001),
		GasLimit: 100000,
	}
	factory := NewFactory(cfg, &mockGasEstimator{})

	// Without contract address - should fail
	_, err := factory.CreateBuilder(config.ModeContractCall)
	if err == nil {
		t.Error("CreateBuilder() expected error without contract address")
	}

	// With contract address - should succeed
	builder, err := factory.CreateBuilder(config.ModeContractCall, WithContractAddress(common.HexToAddress(testContractAddr)))
	if err != nil {
		t.Fatalf("CreateBuilder() error: %v", err)
	}

	if builder.Name() != "CONTRACT_CALL" {
		t.Errorf("Builder.Name() = %s, want CONTRACT_CALL", builder.Name())
	}
}

func TestFactory_CreateBuilder_ERC20_RequiresToken(t *testing.T) {
	cfg := &BuilderConfig{
		ChainID:  big.NewInt(1001),
		GasLimit: 100000,
	}
	factory := NewFactory(cfg, &mockGasEstimator{})

	// Without token address - should fail
	_, err := factory.CreateBuilder(config.ModeERC20Transfer)
	if err == nil {
		t.Error("CreateBuilder() expected error without token address")
	}

	// With token address - should succeed
	builder, err := factory.CreateBuilder(config.ModeERC20Transfer, WithTokenAddress(common.HexToAddress(testTokenAddr)))
	if err != nil {
		t.Fatalf("CreateBuilder() error: %v", err)
	}

	if builder.Name() != "ERC20_TRANSFER" {
		t.Errorf("Builder.Name() = %s, want ERC20_TRANSFER", builder.Name())
	}
}

func TestFactory_CreateBuilder_ContractDeploy(t *testing.T) {
	cfg := &BuilderConfig{
		ChainID:  big.NewInt(1001),
		GasLimit: 1000000,
	}
	factory := NewFactory(cfg, &mockGasEstimator{})

	builder, err := factory.CreateBuilder(config.ModeContractDeploy)
	if err != nil {
		t.Fatalf("CreateBuilder() error: %v", err)
	}

	if builder.Name() != "CONTRACT_DEPLOY" {
		t.Errorf("Builder.Name() = %s, want CONTRACT_DEPLOY", builder.Name())
	}
}

func TestFactory_CreateBuilder_InvalidMode(t *testing.T) {
	cfg := &BuilderConfig{
		ChainID:  big.NewInt(1001),
		GasLimit: 21000,
	}
	factory := NewFactory(cfg, &mockGasEstimator{})

	_, err := factory.CreateBuilder(config.Mode("INVALID"))
	if err == nil {
		t.Error("CreateBuilder() expected error for invalid mode")
	}
}

func TestBaseBuilder_GetGasSettings(t *testing.T) {
	tests := []struct {
		name      string
		config    *BuilderConfig
		estimator GasEstimator
		wantErr   bool
	}{
		{
			name: "with config values",
			config: &BuilderConfig{
				ChainID:   big.NewInt(1),
				GasTipCap: big.NewInt(100000000),
				GasFeeCap: big.NewInt(1000000000),
			},
			estimator: nil,
			wantErr:   false,
		},
		{
			name: "with estimator",
			config: &BuilderConfig{
				ChainID: big.NewInt(1),
			},
			estimator: &mockGasEstimator{
				gasPrice:  big.NewInt(1000000000),
				gasTipCap: big.NewInt(100000000),
			},
			wantErr: false,
		},
		{
			name: "tip cap adjustment",
			config: &BuilderConfig{
				ChainID:   big.NewInt(1),
				GasTipCap: big.NewInt(2000000000), // Higher than gasFeeCap
				GasFeeCap: big.NewInt(1000000000),
			},
			estimator: nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBaseBuilder(tt.config, tt.estimator)
			gasTipCap, gasFeeCap, err := builder.GetGasSettings(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetGasSettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if gasTipCap == nil {
					t.Error("gasTipCap is nil")
				}
				if gasFeeCap == nil {
					t.Error("gasFeeCap is nil")
				}
				// Verify gasTipCap <= gasFeeCap
				if gasTipCap != nil && gasFeeCap != nil && gasTipCap.Cmp(gasFeeCap) > 0 {
					t.Errorf("gasTipCap (%s) > gasFeeCap (%s)", gasTipCap, gasFeeCap)
				}
			}
		})
	}
}

func TestBuilderOptions(t *testing.T) {
	// Test all builder options
	key := newTestKey()
	addr := common.HexToAddress(testContractAddr)
	tokenAddr := common.HexToAddress(testTokenAddr)
	amount := big.NewInt(1000)
	bytecode := []byte{0x60, 0x00, 0x60, 0x00}

	tests := []struct {
		name   string
		option BuilderOption
	}{
		{"WithRecipient", WithRecipient(addr)},
		{"WithFeePayerKey", WithFeePayerKey(key)},
		{"WithContractAddress", WithContractAddress(addr)},
		{"WithTokenAddress", WithTokenAddress(tokenAddr)},
		{"WithBytecode", WithBytecode(bytecode)},
		{"WithMethod", WithMethod("transfer", addr, amount)},
		{"WithABI", WithABI(`[]`)},
		{"WithAmount", WithAmount(amount)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &builderOptions{}
			tt.option(opts)
			// Just verify the option runs without panic
		})
	}
}

func TestPrefixedRlpHash(t *testing.T) {
	// Test that prefixedRlpHash produces consistent results
	data := []interface{}{
		big.NewInt(1),
		uint64(0),
		big.NewInt(100),
	}

	hash1 := prefixedRlpHash(0x02, data)
	hash2 := prefixedRlpHash(0x02, data)

	if hash1 != hash2 {
		t.Error("prefixedRlpHash() not deterministic")
	}

	// Different prefix should produce different hash
	hash3 := prefixedRlpHash(0x16, data)
	if hash1 == hash3 {
		t.Error("Different prefix should produce different hash")
	}
}
