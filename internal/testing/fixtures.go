package testing

import (
	"testing"
	"time"

	"github.com/0xmhha/txhammer/internal/config"
)

// TestConfig creates a valid test configuration
func TestConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		URL:          "http://localhost:8545",
		PrivateKey:   "0x" + TestPrivateKey,
		Mode:         "TRANSFER",
		SubAccounts:  5,
		Transactions: 100,
		BatchSize:    50,
		GasLimit:     21000,
		Timeout:      5 * time.Minute,
	}
}

// TestConfigWithMode creates a test configuration with a specific mode
func TestConfigWithMode(t *testing.T, mode string) *config.Config {
	t.Helper()
	cfg := TestConfig(t)
	cfg.Mode = mode
	return cfg
}

// TestConfigFeeDelegation creates a test configuration for fee delegation mode
func TestConfigFeeDelegation(t *testing.T) *config.Config {
	t.Helper()
	cfg := TestConfig(t)
	cfg.Mode = "FEE_DELEGATION"
	cfg.FeePayerKey = "0xfedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	return cfg
}

// TestConfigContractCall creates a test configuration for contract call mode
func TestConfigContractCall(t *testing.T) *config.Config {
	t.Helper()
	cfg := TestConfig(t)
	cfg.Mode = "CONTRACT_CALL"
	cfg.Contract = "0x1234567890123456789012345678901234567890"
	cfg.Method = "transfer(address,uint256)"
	cfg.GasLimit = 100000
	return cfg
}

// TestConfigERC20 creates a test configuration for ERC20 transfer mode
func TestConfigERC20(t *testing.T) *config.Config {
	t.Helper()
	cfg := TestConfig(t)
	cfg.Mode = "ERC20_TRANSFER"
	cfg.Contract = "0x1234567890123456789012345678901234567890"
	cfg.GasLimit = 65000
	return cfg
}

// TestConfigContractDeploy creates a test configuration for contract deployment mode
func TestConfigContractDeploy(t *testing.T) *config.Config {
	t.Helper()
	cfg := TestConfig(t)
	cfg.Mode = "CONTRACT_DEPLOY"
	cfg.GasLimit = 200000
	return cfg
}

// MinimalConfig creates a minimal valid configuration for quick tests
func MinimalConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		URL:          "http://localhost:8545",
		PrivateKey:   "0x" + TestPrivateKey,
		Mode:         "TRANSFER",
		SubAccounts:  2,
		Transactions: 10,
		BatchSize:    5,
		GasLimit:     21000,
		Timeout:      1 * time.Minute,
	}
}

// InvalidConfigs returns a set of invalid configurations for testing validation
func InvalidConfigs(t *testing.T) map[string]*config.Config {
	t.Helper()
	return map[string]*config.Config{
		"missing_url": {
			PrivateKey:   "0x" + TestPrivateKey,
			Mode:         "TRANSFER",
			SubAccounts:  5,
			Transactions: 100,
			BatchSize:    50,
			GasLimit:     21000,
		},
		"invalid_url": {
			URL:          "invalid-url",
			PrivateKey:   "0x" + TestPrivateKey,
			Mode:         "TRANSFER",
			SubAccounts:  5,
			Transactions: 100,
			BatchSize:    50,
			GasLimit:     21000,
		},
		"missing_credentials": {
			URL:          "http://localhost:8545",
			Mode:         "TRANSFER",
			SubAccounts:  5,
			Transactions: 100,
			BatchSize:    50,
			GasLimit:     21000,
		},
		"invalid_private_key": {
			URL:          "http://localhost:8545",
			PrivateKey:   "invalid-key",
			Mode:         "TRANSFER",
			SubAccounts:  5,
			Transactions: 100,
			BatchSize:    50,
			GasLimit:     21000,
		},
		"invalid_mode": {
			URL:          "http://localhost:8545",
			PrivateKey:   "0x" + TestPrivateKey,
			Mode:         "INVALID_MODE",
			SubAccounts:  5,
			Transactions: 100,
			BatchSize:    50,
			GasLimit:     21000,
		},
		"zero_sub_accounts": {
			URL:          "http://localhost:8545",
			PrivateKey:   "0x" + TestPrivateKey,
			Mode:         "TRANSFER",
			SubAccounts:  0,
			Transactions: 100,
			BatchSize:    50,
			GasLimit:     21000,
		},
		"zero_transactions": {
			URL:          "http://localhost:8545",
			PrivateKey:   "0x" + TestPrivateKey,
			Mode:         "TRANSFER",
			SubAccounts:  5,
			Transactions: 0,
			BatchSize:    50,
			GasLimit:     21000,
		},
		"fee_delegation_no_payer": {
			URL:          "http://localhost:8545",
			PrivateKey:   "0x" + TestPrivateKey,
			Mode:         "FEE_DELEGATION",
			SubAccounts:  5,
			Transactions: 100,
			BatchSize:    50,
			GasLimit:     21000,
		},
		"contract_call_no_address": {
			URL:          "http://localhost:8545",
			PrivateKey:   "0x" + TestPrivateKey,
			Mode:         "CONTRACT_CALL",
			SubAccounts:  5,
			Transactions: 100,
			BatchSize:    50,
			GasLimit:     100000,
		},
		"contract_call_no_method": {
			URL:          "http://localhost:8545",
			PrivateKey:   "0x" + TestPrivateKey,
			Mode:         "CONTRACT_CALL",
			Contract:     "0x1234567890123456789012345678901234567890",
			SubAccounts:  5,
			Transactions: 100,
			BatchSize:    50,
			GasLimit:     100000,
		},
	}
}
