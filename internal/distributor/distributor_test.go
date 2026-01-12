package distributor

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	testPrivateKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

// mustParseBigInt parses a string to big.Int, panics on error
func mustParseBigInt(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("failed to parse big int: " + s)
	}
	return n
}

// mockClient implements Client interface for testing
type mockClient struct {
	balances     map[common.Address]*big.Int
	nonces       map[common.Address]uint64
	gasPrice     *big.Int
	gasTipCap    *big.Int
	chainID      *big.Int
	sentTxs      []*types.Transaction
	balanceErr   error
	nonceErr     error
	sendTxErr    error
	gasPriceErr  error
	gasTipCapErr error
	chainIDErr   error
}

func newMockClient() *mockClient {
	return &mockClient{
		balances:  make(map[common.Address]*big.Int),
		nonces:    make(map[common.Address]uint64),
		gasPrice:  big.NewInt(1000000000),  // 1 Gwei
		gasTipCap: big.NewInt(100000000),   // 0.1 Gwei
		chainID:   big.NewInt(1001),
		sentTxs:   make([]*types.Transaction, 0),
	}
}

func (m *mockClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	if m.balanceErr != nil {
		return nil, m.balanceErr
	}
	if balance, ok := m.balances[account]; ok {
		return balance, nil
	}
	return big.NewInt(0), nil
}

func (m *mockClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	if m.nonceErr != nil {
		return 0, m.nonceErr
	}
	return m.nonces[account], nil
}

func (m *mockClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	if m.gasPriceErr != nil {
		return nil, m.gasPriceErr
	}
	return m.gasPrice, nil
}

func (m *mockClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	if m.gasTipCapErr != nil {
		return nil, m.gasTipCapErr
	}
	return m.gasTipCap, nil
}

func (m *mockClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	if m.sendTxErr != nil {
		return m.sendTxErr
	}
	m.sentTxs = append(m.sentTxs, tx)
	// Update balance of recipient (simulate tx)
	if tx.To() != nil {
		if _, ok := m.balances[*tx.To()]; !ok {
			m.balances[*tx.To()] = big.NewInt(0)
		}
		m.balances[*tx.To()] = new(big.Int).Add(m.balances[*tx.To()], tx.Value())
	}
	return nil
}

func (m *mockClient) ChainID(ctx context.Context) (*big.Int, error) {
	if m.chainIDErr != nil {
		return nil, m.chainIDErr
	}
	return m.chainID, nil
}

func newTestKey() (*ecdsa.PrivateKey, common.Address) {
	key, _ := crypto.HexToECDSA(testPrivateKey)
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return key, addr
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.GasPerTx != 21000 {
		t.Errorf("GasPerTx = %d, want 21000", cfg.GasPerTx)
	}
	if cfg.TxsPerAccount != 10 {
		t.Errorf("TxsPerAccount = %d, want 10", cfg.TxsPerAccount)
	}
	if cfg.GasPrice.Cmp(big.NewInt(1000000000)) != 0 {
		t.Errorf("GasPrice = %s, want 1000000000", cfg.GasPrice.String())
	}
	if cfg.BufferPercent != 20 {
		t.Errorf("BufferPercent = %d, want 20", cfg.BufferPercent)
	}
}

func TestConfig_CalculateRequiredFund(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		wantFunc func(*big.Int) bool
	}{
		{
			name: "default config",
			config: &Config{
				GasPerTx:      21000,
				TxsPerAccount: 10,
				GasPrice:      big.NewInt(1000000000),
				BufferPercent: 20,
			},
			wantFunc: func(result *big.Int) bool {
				// 21000 * 10 * 1000000000 * 1.2 = 252000000000000 (252000 Gwei)
				expected := big.NewInt(252000000000000)
				return result.Cmp(expected) == 0
			},
		},
		{
			name: "no buffer",
			config: &Config{
				GasPerTx:      21000,
				TxsPerAccount: 10,
				GasPrice:      big.NewInt(1000000000),
				BufferPercent: 0,
			},
			wantFunc: func(result *big.Int) bool {
				// 21000 * 10 * 1000000000 = 210000000000000 (210000 Gwei)
				expected := big.NewInt(210000000000000)
				return result.Cmp(expected) == 0
			},
		},
		{
			name: "higher gas limit",
			config: &Config{
				GasPerTx:      100000,
				TxsPerAccount: 5,
				GasPrice:      big.NewInt(2000000000),
				BufferPercent: 10,
			},
			wantFunc: func(result *big.Int) bool {
				// 100000 * 5 * 2000000000 * 1.1 = 1100000000000000
				expected := big.NewInt(1100000000000000)
				return result.Cmp(expected) == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.CalculateRequiredFund()
			if !tt.wantFunc(result) {
				t.Errorf("CalculateRequiredFund() = %s", result.String())
			}
		})
	}
}

func TestNew(t *testing.T) {
	client := newMockClient()

	// With nil config
	d1 := New(client, nil)
	if d1.config == nil {
		t.Error("New() with nil config should use default config")
	}
	if d1.config.GasPerTx != 21000 {
		t.Error("New() should use default config values")
	}

	// With custom config
	customCfg := &Config{
		GasPerTx:      50000,
		TxsPerAccount: 20,
		GasPrice:      big.NewInt(2000000000),
		BufferPercent: 30,
	}
	d2 := New(client, customCfg)
	if d2.config.GasPerTx != 50000 {
		t.Error("New() should use provided config")
	}
}

func TestDistributor_Distribute_AllAccountsFunded(t *testing.T) {
	client := newMockClient()

	// Create sub-accounts that are already funded
	subAccounts := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	}

	// Set high balances for all accounts
	for _, addr := range subAccounts {
		client.balances[addr] = mustParseBigInt("1000000000000000000") // 1 ETH
	}

	cfg := &Config{
		GasPerTx:      21000,
		TxsPerAccount: 10,
		GasPrice:      big.NewInt(1000000000),
		BufferPercent: 20,
	}

	distributor := New(client, cfg)
	masterKey, _ := newTestKey()

	result, err := distributor.Distribute(context.Background(), masterKey, subAccounts)
	if err != nil {
		t.Fatalf("Distribute() error: %v", err)
	}

	// All accounts should be ready
	if len(result.ReadyAccounts) != 2 {
		t.Errorf("ReadyAccounts = %d, want 2", len(result.ReadyAccounts))
	}

	// No unfunded accounts
	if len(result.UnfundedAccounts) != 0 {
		t.Errorf("UnfundedAccounts = %d, want 0", len(result.UnfundedAccounts))
	}

	// No transactions sent
	if result.TxCount != 0 {
		t.Errorf("TxCount = %d, want 0", result.TxCount)
	}

	// No funds distributed
	if result.TotalDistributed.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("TotalDistributed = %s, want 0", result.TotalDistributed.String())
	}
}

func TestDistributor_Distribute_FundAccounts(t *testing.T) {
	client := newMockClient()
	masterKey, masterAddr := newTestKey()

	// Set high balance for master
	client.balances[masterAddr] = mustParseBigInt("10000000000000000000") // 10 ETH

	// Create sub-accounts with zero balance
	subAccounts := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	}

	cfg := &Config{
		GasPerTx:      21000,
		TxsPerAccount: 10,
		GasPrice:      big.NewInt(1000000000),
		BufferPercent: 20,
	}

	distributor := New(client, cfg)

	result, err := distributor.Distribute(context.Background(), masterKey, subAccounts)
	if err != nil {
		t.Fatalf("Distribute() error: %v", err)
	}

	// All accounts should be funded
	if len(result.ReadyAccounts) != 2 {
		t.Errorf("ReadyAccounts = %d, want 2", len(result.ReadyAccounts))
	}

	// Transactions should be sent
	if result.TxCount != 2 {
		t.Errorf("TxCount = %d, want 2", result.TxCount)
	}

	// Total distributed should be positive
	if result.TotalDistributed.Cmp(big.NewInt(0)) <= 0 {
		t.Errorf("TotalDistributed should be positive: %s", result.TotalDistributed.String())
	}

	// Verify transactions were sent to the client
	if len(client.sentTxs) != 2 {
		t.Errorf("sentTxs = %d, want 2", len(client.sentTxs))
	}
}

func TestDistributor_Distribute_InsufficientFunds(t *testing.T) {
	client := newMockClient()
	masterKey, masterAddr := newTestKey()

	// Set very low balance for master
	client.balances[masterAddr] = big.NewInt(1000) // Very low balance

	subAccounts := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
	}

	cfg := &Config{
		GasPerTx:      21000,
		TxsPerAccount: 10,
		GasPrice:      big.NewInt(1000000000),
		BufferPercent: 20,
	}

	distributor := New(client, cfg)

	_, err := distributor.Distribute(context.Background(), masterKey, subAccounts)
	if err == nil {
		t.Error("Distribute() expected error for insufficient funds")
	}

	if err != ErrInsufficientFunds {
		t.Errorf("Distribute() error = %v, want ErrInsufficientFunds", err)
	}
}

func TestDistributor_Distribute_PartialFunding(t *testing.T) {
	client := newMockClient()
	masterKey, masterAddr := newTestKey()

	// Set moderate balance for master (can only fund some accounts)
	cfg := &Config{
		GasPerTx:      21000,
		TxsPerAccount: 10,
		GasPrice:      big.NewInt(1000000000),
		BufferPercent: 20,
	}

	requiredFund := cfg.CalculateRequiredFund()

	// Master has enough for 2 accounts plus gas costs
	masterBalance := new(big.Int).Mul(requiredFund, big.NewInt(3)) // Enough for 2 accounts + gas
	client.balances[masterAddr] = masterBalance

	// Create 5 sub-accounts
	subAccounts := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
		common.HexToAddress("0x4444444444444444444444444444444444444444"),
		common.HexToAddress("0x5555555555555555555555555555555555555555"),
	}

	distributor := New(client, cfg)

	result, err := distributor.Distribute(context.Background(), masterKey, subAccounts)
	if err != nil {
		t.Fatalf("Distribute() error: %v", err)
	}

	// Some accounts should be funded
	if len(result.ReadyAccounts) == 0 {
		t.Error("ReadyAccounts should have some accounts")
	}

	// Some accounts should be unfunded
	if len(result.UnfundedAccounts) == 0 {
		t.Error("UnfundedAccounts should have some accounts")
	}

	// Total should equal original count
	totalAccounts := len(result.ReadyAccounts) + len(result.UnfundedAccounts)
	if totalAccounts != 5 {
		t.Errorf("Total accounts = %d, want 5", totalAccounts)
	}
}

func TestDistributor_GetAccountNonces(t *testing.T) {
	client := newMockClient()

	accounts := []*AccountStatus{
		{Address: common.HexToAddress("0x1111111111111111111111111111111111111111")},
		{Address: common.HexToAddress("0x2222222222222222222222222222222222222222")},
		{Address: common.HexToAddress("0x3333333333333333333333333333333333333333")},
	}

	// Set nonces
	client.nonces[accounts[0].Address] = 5
	client.nonces[accounts[1].Address] = 10
	client.nonces[accounts[2].Address] = 0

	distributor := New(client, nil)

	nonces, err := distributor.GetAccountNonces(context.Background(), accounts)
	if err != nil {
		t.Fatalf("GetAccountNonces() error: %v", err)
	}

	if len(nonces) != 3 {
		t.Fatalf("GetAccountNonces() returned %d nonces, want 3", len(nonces))
	}

	expected := []uint64{5, 10, 0}
	for i, n := range nonces {
		if n != expected[i] {
			t.Errorf("nonces[%d] = %d, want %d", i, n, expected[i])
		}
		// Verify account status is also updated
		if accounts[i].Nonce != expected[i] {
			t.Errorf("accounts[%d].Nonce = %d, want %d", i, accounts[i].Nonce, expected[i])
		}
	}
}

func TestAccountStatus(t *testing.T) {
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	balance := mustParseBigInt("1000000000000000000")
	requiredFund := big.NewInt(500000000000000000)

	status := &AccountStatus{
		Address:      addr,
		Balance:      balance,
		RequiredFund: requiredFund,
		MissingFund:  big.NewInt(0),
		Nonce:        5,
		IsFunded:     true,
	}

	if status.Address != addr {
		t.Errorf("Address = %s, want %s", status.Address.Hex(), addr.Hex())
	}
	if status.Balance.Cmp(balance) != 0 {
		t.Errorf("Balance = %s, want %s", status.Balance.String(), balance.String())
	}
	if status.RequiredFund.Cmp(requiredFund) != 0 {
		t.Errorf("RequiredFund = %s, want %s", status.RequiredFund.String(), requiredFund.String())
	}
	if status.Nonce != 5 {
		t.Errorf("Nonce = %d, want 5", status.Nonce)
	}
	if !status.IsFunded {
		t.Error("IsFunded should be true")
	}
}

func TestDistributionResult(t *testing.T) {
	result := &DistributionResult{
		ReadyAccounts:    make([]*AccountStatus, 5),
		UnfundedAccounts: make([]*AccountStatus, 2),
		TotalDistributed: mustParseBigInt("1000000000000000000"),
		TxCount:          5,
	}

	if len(result.ReadyAccounts) != 5 {
		t.Errorf("ReadyAccounts length = %d, want 5", len(result.ReadyAccounts))
	}
	if len(result.UnfundedAccounts) != 2 {
		t.Errorf("UnfundedAccounts length = %d, want 2", len(result.UnfundedAccounts))
	}
	if result.TxCount != 5 {
		t.Errorf("TxCount = %d, want 5", result.TxCount)
	}
}

func TestDistributor_Distribute_MixedAccounts(t *testing.T) {
	client := newMockClient()
	masterKey, masterAddr := newTestKey()

	// Set high balance for master
	client.balances[masterAddr] = mustParseBigInt("10000000000000000000") // 10 ETH

	cfg := &Config{
		GasPerTx:      21000,
		TxsPerAccount: 10,
		GasPrice:      big.NewInt(1000000000),
		BufferPercent: 20,
	}

	// Some accounts already funded, some not
	subAccounts := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	}

	// First account is already funded
	client.balances[subAccounts[0]] = mustParseBigInt("1000000000000000000")

	distributor := New(client, cfg)

	result, err := distributor.Distribute(context.Background(), masterKey, subAccounts)
	if err != nil {
		t.Fatalf("Distribute() error: %v", err)
	}

	// All accounts should be ready
	if len(result.ReadyAccounts) != 3 {
		t.Errorf("ReadyAccounts = %d, want 3", len(result.ReadyAccounts))
	}

	// Only 2 transactions should be sent (for unfunded accounts)
	if result.TxCount != 2 {
		t.Errorf("TxCount = %d, want 2", result.TxCount)
	}
}

func TestErrors(t *testing.T) {
	if ErrInsufficientFunds.Error() != "insufficient distributor funds" {
		t.Errorf("ErrInsufficientFunds message incorrect")
	}
	if ErrNoAccountsToFund.Error() != "no accounts to fund" {
		t.Errorf("ErrNoAccountsToFund message incorrect")
	}
}
