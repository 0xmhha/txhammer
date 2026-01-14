package txbuilder

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/schollz/progressbar/v3"

	"github.com/0xmhha/txhammer/internal/util/progress"
)

// SimpleStorageBytecode is a simple storage contract bytecode for testing
// contract SimpleStorage { uint256 value; function set(uint256 v) { value = v; } function get() view returns (uint256) { return value; } }
const SimpleStorageBytecode = "608060405234801561001057600080fd5b5060c78061001f6000396000f3fe6080604052348015600f57600080fd5b506004361060325760003560e01c806360fe47b11460375780636d4ce63c146049575b600080fd5b60476042366004605e565b600055565b005b60005460405190815260200160405180910390f35b600060208284031215606f57600080fd5b503591905056fea264697066735822122041c6fd36c2a89c8d6d6ee3b8d14a6a05a4f7a6f25c6e4a7b3c8d9e0f1a2b3c4d564736f6c63430008130033"

// ContractDeployBuilder builds contract deployment transactions
type ContractDeployBuilder struct {
	*BaseBuilder
	bytecode []byte
}

// NewContractDeployBuilder creates a new contract deploy builder
func NewContractDeployBuilder(config *BuilderConfig, estimator GasEstimator) *ContractDeployBuilder {
	// Use simple storage contract by default
	bytecode := common.FromHex(SimpleStorageBytecode)
	return &ContractDeployBuilder{
		BaseBuilder: NewBaseBuilder(config, estimator),
		bytecode:    bytecode,
	}
}

// WithBytecode sets custom contract bytecode
func (b *ContractDeployBuilder) WithBytecode(bytecode []byte) *ContractDeployBuilder {
	b.bytecode = bytecode
	return b
}

// Name returns the builder name
func (b *ContractDeployBuilder) Name() string {
	return "CONTRACT_DEPLOY"
}

// EstimateGas estimates gas for contract deployment
func (b *ContractDeployBuilder) EstimateGas(_ context.Context) (uint64, error) {
	// Contract deployment needs more gas than simple transfer
	// This is a rough estimate; actual gas depends on bytecode size
	return 200000, nil
}

// Build creates contract deployment transactions
func (b *ContractDeployBuilder) Build(ctx context.Context, keys []*ecdsa.PrivateKey, nonces []uint64, count int) ([]*SignedTx, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys provided")
	}
	if len(keys) != len(nonces) {
		return nil, fmt.Errorf("keys and nonces length mismatch")
	}

	gasTipCap, gasFeeCap, err := b.GetGasSettings(ctx)
	if err != nil {
		return nil, err
	}

	gasLimit := b.config.GasLimit
	if gasLimit == 0 {
		gasLimit = 200000
	}

	distribution := DistributeTransactions(len(keys), count)

	totalTxs := 0
	for _, n := range distribution {
		totalTxs += n
	}

	fmt.Printf("\nBuilding Contract Deploy Transactions\n\n")
	bar := progressbar.Default(int64(totalTxs), "txs built")

	signedTxs := make([]*SignedTx, 0, totalTxs)

	for accountIdx, txCount := range distribution {
		key := keys[accountIdx]
		nonce := nonces[accountIdx]
		from := crypto.PubkeyToAddress(key.PublicKey)

		for i := 0; i < txCount; i++ {
			// Contract deployment: to = nil
			tx := types.NewTx(&types.DynamicFeeTx{
				ChainID:   b.config.ChainID,
				Nonce:     nonce,
				GasTipCap: gasTipCap,
				GasFeeCap: gasFeeCap,
				Gas:       gasLimit,
				To:        nil, // Contract creation
				Value:     big.NewInt(0),
				Data:      b.bytecode,
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

	fmt.Printf("\n[OK] Successfully built %d contract deploy transactions\n", len(signedTxs))
	return signedTxs, nil
}

// ContractCallBuilder builds contract call transactions
type ContractCallBuilder struct {
	*BaseBuilder
	contractAddr common.Address
	methodSig    string
	methodArgs   []interface{}
	parsedABI    abi.ABI
}

// NewContractCallBuilder creates a new contract call builder
func NewContractCallBuilder(config *BuilderConfig, estimator GasEstimator, contractAddr common.Address) *ContractCallBuilder {
	return &ContractCallBuilder{
		BaseBuilder:  NewBaseBuilder(config, estimator),
		contractAddr: contractAddr,
	}
}

// WithMethod sets the method to call
func (b *ContractCallBuilder) WithMethod(methodSig string, args ...interface{}) *ContractCallBuilder {
	b.methodSig = methodSig
	b.methodArgs = args
	return b
}

// WithABI sets the contract ABI
func (b *ContractCallBuilder) WithABI(abiJSON string) (*ContractCallBuilder, error) {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}
	b.parsedABI = parsed
	return b, nil
}

// Name returns the builder name
func (b *ContractCallBuilder) Name() string {
	return "CONTRACT_CALL"
}

// EstimateGas estimates gas for contract call
func (b *ContractCallBuilder) EstimateGas(_ context.Context) (uint64, error) {
	// Contract calls typically need more gas
	return 100000, nil
}

// Build creates contract call transactions
func (b *ContractCallBuilder) Build(ctx context.Context, keys []*ecdsa.PrivateKey, nonces []uint64, count int) ([]*SignedTx, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys provided")
	}
	if len(keys) != len(nonces) {
		return nil, fmt.Errorf("keys and nonces length mismatch")
	}
	if b.contractAddr == (common.Address{}) {
		return nil, fmt.Errorf("contract address is required")
	}

	// Build call data
	callData, err := b.buildCallData()
	if err != nil {
		return nil, err
	}

	gasTipCap, gasFeeCap, err := b.GetGasSettings(ctx)
	if err != nil {
		return nil, err
	}

	gasLimit := b.config.GasLimit
	if gasLimit == 0 {
		gasLimit = 100000
	}

	distribution := DistributeTransactions(len(keys), count)

	totalTxs := 0
	for _, n := range distribution {
		totalTxs += n
	}

	fmt.Printf("\nBuilding Contract Call Transactions\n\n")
	fmt.Printf("Contract: %s\n", b.contractAddr.Hex())
	fmt.Printf("Method: %s\n", b.methodSig)
	bar := progressbar.Default(int64(totalTxs), "txs built")

	signedTxs := make([]*SignedTx, 0, totalTxs)

	for accountIdx, txCount := range distribution {
		key := keys[accountIdx]
		nonce := nonces[accountIdx]
		from := crypto.PubkeyToAddress(key.PublicKey)

		for i := 0; i < txCount; i++ {
			tx := types.NewTx(&types.DynamicFeeTx{
				ChainID:   b.config.ChainID,
				Nonce:     nonce,
				GasTipCap: gasTipCap,
				GasFeeCap: gasFeeCap,
				Gas:       gasLimit,
				To:        &b.contractAddr,
				Value:     big.NewInt(0),
				Data:      callData,
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

	fmt.Printf("\n[OK] Successfully built %d contract call transactions\n", len(signedTxs))
	return signedTxs, nil
}

// buildCallData builds the call data from method signature and arguments
func (b *ContractCallBuilder) buildCallData() ([]byte, error) {
	if b.methodSig == "" {
		return nil, fmt.Errorf("method signature is required")
	}

	// If we have a parsed ABI, use it
	if len(b.parsedABI.Methods) > 0 {
		// Extract method name from signature
		methodName := strings.Split(b.methodSig, "(")[0]
		method, exists := b.parsedABI.Methods[methodName]
		if !exists {
			return nil, fmt.Errorf("method %s not found in ABI", methodName)
		}
		return b.parsedABI.Pack(method.Name, b.methodArgs...)
	}

	// Otherwise, compute method selector from signature
	// methodSig format: "transfer(address,uint256)"
	selector := crypto.Keccak256([]byte(b.methodSig))[:4]

	// For simple cases without ABI, just return selector
	// Complex argument encoding requires full ABI
	if len(b.methodArgs) == 0 {
		return selector, nil
	}

	return nil, fmt.Errorf("full ABI required for methods with arguments")
}
