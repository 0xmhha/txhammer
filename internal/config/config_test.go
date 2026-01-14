package config

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with private key",
			config: &Config{
				URL:          "http://localhost:8545",
				PrivateKey:   "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Mode:         "TRANSFER",
				SubAccounts:  10,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     21000,
			},
			wantErr: false,
		},
		{
			name: "valid config with mnemonic",
			config: &Config{
				URL:          "http://localhost:8545",
				Mnemonic:     "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
				Mode:         "TRANSFER",
				SubAccounts:  10,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     21000,
			},
			wantErr: false,
		},
		{
			name: "valid websocket url",
			config: &Config{
				URL:          "ws://localhost:8546",
				PrivateKey:   "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Mode:         "TRANSFER",
				SubAccounts:  10,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     21000,
			},
			wantErr: false,
		},
		{
			name: "missing url",
			config: &Config{
				PrivateKey:   "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Mode:         "TRANSFER",
				SubAccounts:  10,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     21000,
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "invalid url format",
			config: &Config{
				URL:          "invalid-url",
				PrivateKey:   "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Mode:         "TRANSFER",
				SubAccounts:  10,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     21000,
			},
			wantErr: true,
			errMsg:  "url must be a valid HTTP or WebSocket URL",
		},
		{
			name: "missing credentials",
			config: &Config{
				URL:          "http://localhost:8545",
				Mode:         "TRANSFER",
				SubAccounts:  10,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     21000,
			},
			wantErr: true,
			errMsg:  "either private-key or mnemonic is required",
		},
		{
			name: "invalid private key format",
			config: &Config{
				URL:          "http://localhost:8545",
				PrivateKey:   "invalid-key",
				Mode:         "TRANSFER",
				SubAccounts:  10,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     21000,
			},
			wantErr: true,
			errMsg:  "private-key must be a valid 64-character hex string",
		},
		{
			name: "invalid mode",
			config: &Config{
				URL:          "http://localhost:8545",
				PrivateKey:   "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Mode:         "INVALID_MODE",
				SubAccounts:  10,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     21000,
			},
			wantErr: true,
			errMsg:  "invalid mode",
		},
		{
			name: "fee delegation without fee payer key",
			config: &Config{
				URL:          "http://localhost:8545",
				PrivateKey:   "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Mode:         "FEE_DELEGATION",
				SubAccounts:  10,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     21000,
			},
			wantErr: true,
			errMsg:  "fee-payer-key is required for FEE_DELEGATION mode",
		},
		{
			name: "valid fee delegation config",
			config: &Config{
				URL:          "http://localhost:8545",
				PrivateKey:   "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				FeePayerKey:  "0xfedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
				Mode:         "FEE_DELEGATION",
				SubAccounts:  10,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     21000,
			},
			wantErr: false,
		},
		{
			name: "contract call without contract address",
			config: &Config{
				URL:          "http://localhost:8545",
				PrivateKey:   "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Mode:         "CONTRACT_CALL",
				SubAccounts:  10,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     100000,
			},
			wantErr: true,
			errMsg:  "contract address is required",
		},
		{
			name: "contract call without method",
			config: &Config{
				URL:          "http://localhost:8545",
				PrivateKey:   "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Mode:         "CONTRACT_CALL",
				Contract:     "0x1234567890123456789012345678901234567890",
				SubAccounts:  10,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     100000,
			},
			wantErr: true,
			errMsg:  "method is required for CONTRACT_CALL mode",
		},
		{
			name: "zero sub-accounts",
			config: &Config{
				URL:          "http://localhost:8545",
				PrivateKey:   "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Mode:         "TRANSFER",
				SubAccounts:  0,
				Transactions: 100,
				BatchSize:    50,
				GasLimit:     21000,
			},
			wantErr: true,
			errMsg:  "sub-accounts must be greater than 0",
		},
		{
			name: "zero transactions",
			config: &Config{
				URL:          "http://localhost:8545",
				PrivateKey:   "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Mode:         "TRANSFER",
				SubAccounts:  10,
				Transactions: 0,
				BatchSize:    50,
				GasLimit:     21000,
			},
			wantErr: true,
			errMsg:  "transactions must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					// Check if error message contains expected substring
					if !contains(err.Error(), tt.errMsg) {
						t.Errorf("Config.Validate() error = %v, want error containing %v", err, tt.errMsg)
					}
				}
			}
		})
	}
}

func TestConfig_GetMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected Mode
	}{
		{"transfer lowercase", "transfer", ModeTransfer},
		{"transfer uppercase", "TRANSFER", ModeTransfer},
		{"transfer mixed case", "Transfer", ModeTransfer},
		{"fee delegation", "fee_delegation", ModeFeeDelegation},
		{"contract deploy", "CONTRACT_DEPLOY", ModeContractDeploy},
		{"contract call", "contract_call", ModeContractCall},
		{"erc20 transfer", "ERC20_TRANSFER", ModeERC20Transfer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Mode: tt.mode}
			if got := cfg.GetMode(); got != tt.expected {
				t.Errorf("Config.GetMode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_IsWebSocket(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"http url", "http://localhost:8545", false},
		{"https url", "https://localhost:8545", false},
		{"ws url", "ws://localhost:8546", true},
		{"wss url", "wss://localhost:8546", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{URL: tt.url}
			if got := cfg.IsWebSocket(); got != tt.expected {
				t.Errorf("Config.IsWebSocket() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_DefaultTimeout(t *testing.T) {
	cfg := &Config{
		URL:          "http://localhost:8545",
		PrivateKey:   "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Mode:         "TRANSFER",
		SubAccounts:  10,
		Transactions: 100,
		BatchSize:    50,
		GasLimit:     21000,
		Timeout:      0,
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Expected default timeout of 5m, got %v", cfg.Timeout)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (s != "" && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
