package testing

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// MockClient is a mock implementation of the RPC client for testing
type MockClient struct {
	mu sync.RWMutex

	// Configurable return values
	ChainIDValue       *big.Int
	BlockNumberValue   uint64
	BalanceValue       *big.Int
	NonceValue         uint64
	GasPriceValue      *big.Int
	GasTipCapValue     *big.Int
	EstimateGasValue   uint64
	BlockGasLimitValue uint64

	// Error responses
	ChainIDError         error
	BlockNumberError     error
	BalanceError         error
	NonceError           error
	GasPriceError        error
	GasTipCapError       error
	EstimateGasError     error
	SendTransactionError error
	ReceiptError         error

	// Receipts storage
	Receipts map[common.Hash]*types.Receipt

	// Sent transactions tracking
	SentTransactions []*types.Transaction
	SentRawTxs       [][]byte

	// Call counters
	CallCounts map[string]int
}

// NewMockClient creates a new mock client with default values
func NewMockClient() *MockClient {
	return &MockClient{
		ChainIDValue:       big.NewInt(1337),
		BlockNumberValue:   1000,
		BalanceValue:       big.NewInt(1e18), // 1 ETH
		NonceValue:         0,
		GasPriceValue:      big.NewInt(1e9), // 1 Gwei
		GasTipCapValue:     big.NewInt(1e9),
		EstimateGasValue:   21000,
		BlockGasLimitValue: 30000000,
		Receipts:           make(map[common.Hash]*types.Receipt),
		SentTransactions:   make([]*types.Transaction, 0),
		SentRawTxs:         make([][]byte, 0),
		CallCounts:         make(map[string]int),
	}
}

func (m *MockClient) incrementCallCount(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallCounts[method]++
}

// GetCallCount returns the number of times a method was called
func (m *MockClient) GetCallCount(method string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.CallCounts[method]
}

// Close closes the mock client
func (m *MockClient) Close() {
	m.incrementCallCount("Close")
}

// ChainID returns the configured chain ID
func (m *MockClient) ChainID(ctx context.Context) (*big.Int, error) {
	m.incrementCallCount("ChainID")
	if m.ChainIDError != nil {
		return nil, m.ChainIDError
	}
	return m.ChainIDValue, nil
}

// BlockNumber returns the configured block number
func (m *MockClient) BlockNumber(ctx context.Context) (uint64, error) {
	m.incrementCallCount("BlockNumber")
	if m.BlockNumberError != nil {
		return 0, m.BlockNumberError
	}
	return m.BlockNumberValue, nil
}

// BlockByNumber returns a mock block
func (m *MockClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	m.incrementCallCount("BlockByNumber")
	// Return a minimal block for testing
	header := &types.Header{
		Number:   number,
		GasLimit: m.BlockGasLimitValue,
		GasUsed:  0,
	}
	return types.NewBlockWithHeader(header), nil
}

// BalanceAt returns the configured balance
func (m *MockClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	m.incrementCallCount("BalanceAt")
	if m.BalanceError != nil {
		return nil, m.BalanceError
	}
	return m.BalanceValue, nil
}

// PendingNonceAt returns the configured nonce
func (m *MockClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	m.incrementCallCount("PendingNonceAt")
	if m.NonceError != nil {
		return 0, m.NonceError
	}
	return m.NonceValue, nil
}

// SuggestGasPrice returns the configured gas price
func (m *MockClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	m.incrementCallCount("SuggestGasPrice")
	if m.GasPriceError != nil {
		return nil, m.GasPriceError
	}
	return m.GasPriceValue, nil
}

// SuggestGasTipCap returns the configured gas tip cap
func (m *MockClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	m.incrementCallCount("SuggestGasTipCap")
	if m.GasTipCapError != nil {
		return nil, m.GasTipCapError
	}
	return m.GasTipCapValue, nil
}

// EstimateGas returns the configured gas estimate
func (m *MockClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	m.incrementCallCount("EstimateGas")
	if m.EstimateGasError != nil {
		return 0, m.EstimateGasError
	}
	return m.EstimateGasValue, nil
}

// SendTransaction stores the transaction and returns configured error
func (m *MockClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	m.incrementCallCount("SendTransaction")
	m.mu.Lock()
	m.SentTransactions = append(m.SentTransactions, tx)
	m.mu.Unlock()
	return m.SendTransactionError
}

// TransactionReceipt returns the receipt for a transaction
func (m *MockClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	m.incrementCallCount("TransactionReceipt")
	if m.ReceiptError != nil {
		return nil, m.ReceiptError
	}
	m.mu.RLock()
	receipt, ok := m.Receipts[txHash]
	m.mu.RUnlock()
	if !ok {
		return nil, ethereum.NotFound
	}
	return receipt, nil
}

// HeaderByNumber returns a mock header
func (m *MockClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	m.incrementCallCount("HeaderByNumber")
	return &types.Header{
		Number:   number,
		GasLimit: m.BlockGasLimitValue,
	}, nil
}

// BatchCall is a mock implementation of batch RPC calls
func (m *MockClient) BatchCall(b []rpc.BatchElem) error {
	m.incrementCallCount("BatchCall")
	return nil
}

// SendRawTransaction stores the raw transaction
func (m *MockClient) SendRawTransaction(ctx context.Context, rawTx []byte) (common.Hash, error) {
	m.incrementCallCount("SendRawTransaction")
	m.mu.Lock()
	m.SentRawTxs = append(m.SentRawTxs, rawTx)
	m.mu.Unlock()
	if m.SendTransactionError != nil {
		return common.Hash{}, m.SendTransactionError
	}
	return common.BytesToHash(rawTx[:32]), nil
}

// BatchSendRawTransactions sends multiple raw transactions
func (m *MockClient) BatchSendRawTransactions(ctx context.Context, rawTxs [][]byte) ([]common.Hash, error) {
	m.incrementCallCount("BatchSendRawTransactions")
	m.mu.Lock()
	m.SentRawTxs = append(m.SentRawTxs, rawTxs...)
	m.mu.Unlock()
	if m.SendTransactionError != nil {
		return nil, m.SendTransactionError
	}
	hashes := make([]common.Hash, len(rawTxs))
	for i, tx := range rawTxs {
		if len(tx) >= 32 {
			hashes[i] = common.BytesToHash(tx[:32])
		}
	}
	return hashes, nil
}

// GetBlockGasLimit returns the configured block gas limit
func (m *MockClient) GetBlockGasLimit(ctx context.Context, blockNumber uint64) (uint64, error) {
	m.incrementCallCount("GetBlockGasLimit")
	return m.BlockGasLimitValue, nil
}

// GetBlockGasUsed returns mock gas used
func (m *MockClient) GetBlockGasUsed(ctx context.Context, blockNumber uint64) (uint64, error) {
	m.incrementCallCount("GetBlockGasUsed")
	return 0, nil
}

// AddReceipt adds a receipt to the mock storage
func (m *MockClient) AddReceipt(txHash common.Hash, receipt *types.Receipt) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Receipts[txHash] = receipt
}

// CreateSuccessReceipt creates a successful receipt for testing
func CreateSuccessReceipt(txHash common.Hash, blockNumber uint64, gasUsed uint64) *types.Receipt {
	return &types.Receipt{
		Status:      types.ReceiptStatusSuccessful,
		TxHash:      txHash,
		BlockNumber: big.NewInt(int64(blockNumber)),
		GasUsed:     gasUsed,
	}
}

// CreateFailedReceipt creates a failed receipt for testing
func CreateFailedReceipt(txHash common.Hash, blockNumber uint64, gasUsed uint64) *types.Receipt {
	return &types.Receipt{
		Status:      types.ReceiptStatusFailed,
		TxHash:      txHash,
		BlockNumber: big.NewInt(int64(blockNumber)),
		GasUsed:     gasUsed,
	}
}
