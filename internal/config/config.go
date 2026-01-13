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
	ModeLongSender     Mode = "LONG_SENDER"
	ModeAnalyzeBlocks  Mode = "ANALYZE_BLOCKS"
	ModeERC721Mint     Mode = "ERC721_MINT"
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

	// Prometheus metrics
	MetricsEnabled bool
	MetricsPort    int

	// Long Sender mode
	Duration  time.Duration
	TargetTPS float64
	Workers   int

	// Block Analyzer mode
	BlockStart int64
	BlockEnd   int64
	BlockRange int64

	// ERC721 Mint mode
	NFTName   string
	NFTSymbol string
	TokenURI  string
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

	// Validate account credentials (not required for ANALYZE_BLOCKS mode)
	mode := Mode(strings.ToUpper(c.Mode))
	if mode != ModeAnalyzeBlocks {
		if c.PrivateKey == "" && c.Mnemonic == "" {
			return errors.New("either private-key or mnemonic is required")
		}
		if c.PrivateKey != "" && !hexKeyRegex.MatchString(c.PrivateKey) {
			return errors.New("private-key must be a valid 64-character hex string with 0x prefix")
		}
	}

	// Validate mode (mode already declared above for account validation)
	switch mode {
	case ModeTransfer, ModeFeeDelegation, ModeContractDeploy, ModeContractCall, ModeERC20Transfer,
		ModeLongSender, ModeAnalyzeBlocks, ModeERC721Mint:
		// Valid modes
	default:
		return errors.New("invalid mode: must be TRANSFER, FEE_DELEGATION, CONTRACT_DEPLOY, CONTRACT_CALL, ERC20_TRANSFER, LONG_SENDER, ANALYZE_BLOCKS, or ERC721_MINT")
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

	// Validate numeric values (not required for ANALYZE_BLOCKS mode)
	if mode != ModeAnalyzeBlocks {
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
	}

	// Set default timeout
	if c.Timeout == 0 {
		c.Timeout = 5 * time.Minute
	}

	// Validate Long Sender mode requirements
	if mode == ModeLongSender {
		if c.TargetTPS <= 0 {
			c.TargetTPS = 100 // default TPS
		}
		if c.Workers <= 0 {
			c.Workers = 10 // default workers
		}
	}

	// Validate Block Analyzer mode requirements
	if mode == ModeAnalyzeBlocks {
		if c.BlockStart == 0 && c.BlockEnd == 0 && c.BlockRange == 0 {
			c.BlockRange = 100 // default to last 100 blocks
		}
		if c.BlockStart > 0 && c.BlockEnd > 0 && c.BlockStart > c.BlockEnd {
			return errors.New("block-start must be less than or equal to block-end")
		}
	}

	// Validate ERC721 Mint mode requirements
	if mode == ModeERC721Mint {
		if c.NFTName == "" {
			c.NFTName = "TxHammerNFT"
		}
		if c.NFTSymbol == "" {
			c.NFTSymbol = "TXHNFT"
		}
		if c.TokenURI == "" {
			c.TokenURI = "https://txhammer.io/nft/"
		}
	}

	// Set default metrics port
	if c.MetricsEnabled && c.MetricsPort == 0 {
		c.MetricsPort = 9090
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
