package distributor

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/schollz/progressbar/v3"
)

var (
	ErrInsufficientFunds = errors.New("insufficient distributor funds")
	ErrNoAccountsToFund  = errors.New("no accounts to fund")
)

// Client interface for blockchain operations
type Client interface {
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	ChainID(ctx context.Context) (*big.Int, error)
}

// Distributor manages fund distribution from master to sub-accounts
type Distributor struct {
	client  Client
	config  *Config
	chainID *big.Int
}

// New creates a new Distributor instance
func New(client Client, config *Config) *Distributor {
	if config == nil {
		config = DefaultConfig()
	}
	return &Distributor{
		client: client,
		config: config,
	}
}

// Distribute distributes funds from the master account to sub-accounts
func (d *Distributor) Distribute(
	ctx context.Context,
	masterKey *ecdsa.PrivateKey,
	subAccounts []common.Address,
) (*DistributionResult, error) {
	fmt.Printf("\nStarting Fund Distribution\n\n")

	// Get chain ID if not set
	if d.chainID == nil {
		chainID, err := d.client.ChainID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get chain ID: %w", err)
		}
		d.chainID = chainID
	}

	// Calculate required fund per account
	requiredFund := d.config.CalculateRequiredFund()
	fmt.Printf("Required fund per account: %s wei\n", requiredFund.String())
	fmt.Printf("  Gas per tx: %d\n", d.config.GasPerTx)
	fmt.Printf("  Txs per account: %d\n", d.config.TxsPerAccount)
	fmt.Printf("  Buffer: %d%%\n\n", d.config.BufferPercent)

	// Check account balances and identify which need funding
	accountStatuses, err := d.checkBalances(ctx, subAccounts, requiredFund)
	if err != nil {
		return nil, fmt.Errorf("failed to check balances: %w", err)
	}

	// Separate funded and unfunded accounts
	var fundedAccounts, unfundedAccounts []*AccountStatus
	for _, status := range accountStatuses {
		if status.IsFunded {
			fundedAccounts = append(fundedAccounts, status)
		} else {
			unfundedAccounts = append(unfundedAccounts, status)
		}
	}

	// If all accounts are already funded
	if len(unfundedAccounts) == 0 {
		fmt.Printf("[OK] All %d accounts are already funded\n", len(fundedAccounts))
		return &DistributionResult{
			ReadyAccounts:    fundedAccounts,
			UnfundedAccounts: nil,
			TotalDistributed: big.NewInt(0),
			TxCount:          0,
		}, nil
	}

	// Sort unfunded accounts by missing fund (ascending)
	sort.Slice(unfundedAccounts, func(i, j int) bool {
		return unfundedAccounts[i].MissingFund.Cmp(unfundedAccounts[j].MissingFund) < 0
	})

	// Fund the accounts
	result, err := d.fundAccounts(ctx, masterKey, unfundedAccounts)
	if err != nil {
		return nil, err
	}

	// Combine results
	result.ReadyAccounts = append(fundedAccounts, result.ReadyAccounts...)

	return result, nil
}

// checkBalances checks the balance of each account and determines funding needs
func (d *Distributor) checkBalances(
	ctx context.Context,
	accounts []common.Address,
	requiredFund *big.Int,
) ([]*AccountStatus, error) {
	fmt.Printf("Checking balances of %d accounts...\n", len(accounts))
	bar := progressbar.Default(int64(len(accounts)), "checking balances")

	statuses := make([]*AccountStatus, 0, len(accounts))

	for _, addr := range accounts {
		balance, err := d.client.BalanceAt(ctx, addr, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get balance for %s: %w", addr.Hex(), err)
		}

		nonce, err := d.client.PendingNonceAt(ctx, addr)
		if err != nil {
			return nil, fmt.Errorf("failed to get nonce for %s: %w", addr.Hex(), err)
		}

		status := &AccountStatus{
			Address:      addr,
			Balance:      balance,
			RequiredFund: requiredFund,
			Nonce:        nonce,
		}

		// Check if account has enough funds
		if balance.Cmp(requiredFund) >= 0 {
			status.IsFunded = true
			status.MissingFund = big.NewInt(0)
		} else {
			status.IsFunded = false
			status.MissingFund = new(big.Int).Sub(requiredFund, balance)
		}

		statuses = append(statuses, status)
		_ = bar.Add(1)
	}

	fmt.Println()
	return statuses, nil
}

// fundAccounts sends funds to accounts that need it
func (d *Distributor) fundAccounts(
	ctx context.Context,
	masterKey *ecdsa.PrivateKey,
	unfundedAccounts []*AccountStatus,
) (*DistributionResult, error) {
	masterAddr := crypto.PubkeyToAddress(masterKey.PublicKey)

	// Check master balance
	masterBalance, err := d.client.BalanceAt(ctx, masterAddr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get master balance: %w", err)
	}

	fmt.Printf("Master account: %s\n", masterAddr.Hex())
	fmt.Printf("Master balance: %s wei\n\n", masterBalance.String())

	// Get gas price - use config GasPrice if available, otherwise suggest
	var gasPrice *big.Int
	if d.config.GasPrice != nil && d.config.GasPrice.Sign() > 0 {
		// Use configured gas price
		gasPrice = new(big.Int).Set(d.config.GasPrice)
	} else {
		var err error
		gasPrice, err = d.client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to suggest gas price: %w", err)
		}
	}

	// Transfer gas cost (21000 gas for simple transfer)
	transferGas := uint64(21000)
	transferCost := new(big.Int).Mul(gasPrice, big.NewInt(int64(transferGas)))

	// Calculate how many accounts we can fund
	fundableAccounts := make([]*AccountStatus, 0)
	remainingBalance := new(big.Int).Set(masterBalance)
	totalToDistribute := big.NewInt(0)

	for _, account := range unfundedAccounts {
		// Cost = missing fund + transfer gas cost
		cost := new(big.Int).Add(account.MissingFund, transferCost)

		if remainingBalance.Cmp(cost) < 0 {
			// Not enough balance to fund this account
			break
		}

		fundableAccounts = append(fundableAccounts, account)
		remainingBalance.Sub(remainingBalance, cost)
		totalToDistribute.Add(totalToDistribute, account.MissingFund)
	}

	if len(fundableAccounts) == 0 {
		fmt.Printf("[FAIL] Master account cannot fund any sub-accounts\n")
		fmt.Printf("   Master balance: %s wei\n", masterBalance.String())
		fmt.Printf("   Minimum needed: %s wei\n", unfundedAccounts[0].MissingFund.String())
		return nil, ErrInsufficientFunds
	}

	fmt.Printf("Funding %d accounts...\n", len(fundableAccounts))
	bar := progressbar.Default(int64(len(fundableAccounts)), "funding accounts")

	// Get master nonce
	nonce, err := d.client.PendingNonceAt(ctx, masterAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get master nonce: %w", err)
	}

	readyAccounts := make([]*AccountStatus, 0, len(fundableAccounts))
	txCount := 0

	for _, account := range fundableAccounts {
		// Create legacy transaction (type 0) for better compatibility
		tx := types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      transferGas,
			To:       &account.Address,
			Value:    account.MissingFund,
			Data:     nil,
		})

		// Sign transaction with EIP-155 signer
		signer := types.NewEIP155Signer(d.chainID)
		signedTx, err := types.SignTx(tx, signer, masterKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign transfer tx: %w", err)
		}

		// Send transaction
		if err := d.client.SendTransaction(ctx, signedTx); err != nil {
			return nil, fmt.Errorf("failed to send transfer tx to %s: %w", account.Address.Hex(), err)
		}

		nonce++
		txCount++

		// Mark account as funded
		account.IsFunded = true
		account.Balance = new(big.Int).Add(account.Balance, account.MissingFund)
		readyAccounts = append(readyAccounts, account)

		_ = bar.Add(1)

		// Small delay to avoid overwhelming the node
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Printf("\n[OK] Successfully funded %d accounts\n", len(readyAccounts))
	fmt.Printf("   Total distributed: %s wei\n", totalToDistribute.String())

	// Calculate unfunded accounts
	unfunded := make([]*AccountStatus, 0)
	for i := len(fundableAccounts); i < len(unfundedAccounts); i++ {
		unfunded = append(unfunded, unfundedAccounts[i])
	}

	if len(unfunded) > 0 {
		fmt.Printf("   [WARN] %d accounts could not be funded (insufficient master balance)\n", len(unfunded))
	}

	return &DistributionResult{
		ReadyAccounts:    readyAccounts,
		UnfundedAccounts: unfunded,
		TotalDistributed: totalToDistribute,
		TxCount:          txCount,
	}, nil
}

// WaitForFunding waits for all distribution transactions to be confirmed
func (d *Distributor) WaitForFunding(
	ctx context.Context,
	accounts []*AccountStatus,
	timeout time.Duration,
) error {
	fmt.Printf("\nWaiting for funding confirmations...\n")

	deadline := time.Now().Add(timeout)
	bar := progressbar.Default(int64(len(accounts)), "confirming")

	for _, account := range accounts {
		for {
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for funding confirmation")
			}

			balance, err := d.client.BalanceAt(ctx, account.Address, nil)
			if err != nil {
				return fmt.Errorf("failed to check balance: %w", err)
			}

			if balance.Cmp(account.RequiredFund) >= 0 {
				account.Balance = balance
				_ = bar.Add(1)
				break
			}

			time.Sleep(500 * time.Millisecond)
		}
	}

	fmt.Printf("[OK] All funding transactions confirmed\n")
	return nil
}

// GetAccountNonces fetches the current nonce for each account
func (d *Distributor) GetAccountNonces(
	ctx context.Context,
	accounts []*AccountStatus,
) ([]uint64, error) {
	nonces := make([]uint64, len(accounts))

	for i, account := range accounts {
		nonce, err := d.client.PendingNonceAt(ctx, account.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to get nonce for %s: %w", account.Address.Hex(), err)
		}
		nonces[i] = nonce
		account.Nonce = nonce
	}

	return nonces, nil
}
