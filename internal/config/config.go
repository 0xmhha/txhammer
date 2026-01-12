package config

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Mode represents the stress test mode
type Mode string

const (
	ModeTransfer       Mode = "TRANSFER"
	ModeFeeDelegation  Mode = "FEE_DELEGATION"
	ModeContractDeploy Mode = "CONTRACT_DEPLOY"
	ModeContractCall   Mode = "CONTRACT_CALL"
	ModeERC20Transfer  Mode = "ERC20_TRANSFER"
)

// Config holds all configuration for the stress test
type Config struct {
	// RPC connection
	URL string

	// Account configuration
	PrivateKey string
	Mnemonic   string

	// Test configuration
	Mode         string
	SubAccounts  uint64
	Transactions uint64
	BatchSize    uint64

	// Chain configuration
	ChainID  uint64
	GasLimit uint64
	GasPrice string

	// Fee Delegation mode
	FeePayerKey string

	// Contract mode
	Contract string
	Method   string
	Args     string

	// Output
	Output  string
	Verbose bool

	// Advanced
	Timeout   time.Duration
	RateLimit uint64
}

var (
	httpRegex    = regexp.MustCompile(`^https?://`)
	wsRegex      = regexp.MustCompile(`^wss?://`)
	hexKeyRegex  = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
	addressRegex = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
)

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate URL
	if c.URL == "" {
		return errors.New("url is required")
	}
	if !httpRegex.MatchString(c.URL) && !wsRegex.MatchString(c.URL) {
		return errors.New("url must be a valid HTTP or WebSocket URL")
	}

	// Validate account credentials
	if c.PrivateKey == "" && c.Mnemonic == "" {
		return errors.New("either private-key or mnemonic is required")
	}
	if c.PrivateKey != "" && !hexKeyRegex.MatchString(c.PrivateKey) {
		return errors.New("private-key must be a valid 64-character hex string with 0x prefix")
	}

	// Validate mode
	mode := Mode(strings.ToUpper(c.Mode))
	switch mode {
	case ModeTransfer, ModeFeeDelegation, ModeContractDeploy, ModeContractCall, ModeERC20Transfer:
		// Valid modes
	default:
		return errors.New("invalid mode: must be TRANSFER, FEE_DELEGATION, CONTRACT_DEPLOY, CONTRACT_CALL, or ERC20_TRANSFER")
	}

	// Validate Fee Delegation mode requirements
	if mode == ModeFeeDelegation {
		if c.FeePayerKey == "" {
			return errors.New("fee-payer-key is required for FEE_DELEGATION mode")
		}
		if !hexKeyRegex.MatchString(c.FeePayerKey) {
			return errors.New("fee-payer-key must be a valid 64-character hex string with 0x prefix")
		}
	}

	// Validate Contract mode requirements
	if mode == ModeContractCall || mode == ModeERC20Transfer {
		if c.Contract == "" {
			return errors.New("contract address is required for CONTRACT_CALL and ERC20_TRANSFER modes")
		}
		if !addressRegex.MatchString(c.Contract) {
			return errors.New("contract must be a valid 40-character hex address with 0x prefix")
		}
	}

	if mode == ModeContractCall && c.Method == "" {
		return errors.New("method is required for CONTRACT_CALL mode")
	}

	// Validate numeric values
	if c.SubAccounts == 0 {
		return errors.New("sub-accounts must be greater than 0")
	}
	if c.Transactions == 0 {
		return errors.New("transactions must be greater than 0")
	}
	if c.BatchSize == 0 {
		return errors.New("batch size must be greater than 0")
	}
	if c.GasLimit == 0 {
		return errors.New("gas-limit must be greater than 0")
	}

	// Set default timeout
	if c.Timeout == 0 {
		c.Timeout = 5 * time.Minute
	}

	return nil
}

// GetMode returns the parsed mode
func (c *Config) GetMode() Mode {
	return Mode(strings.ToUpper(c.Mode))
}

// IsWebSocket returns true if the URL is a WebSocket URL
func (c *Config) IsWebSocket() bool {
	return wsRegex.MatchString(c.URL)
}
