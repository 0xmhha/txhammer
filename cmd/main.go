package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/0xmhha/txhammer/internal/config"
	"github.com/0xmhha/txhammer/internal/pipeline"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	cfg     = &config.Config{}
	runCfg  = &pipeline.RunConfig{}
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "txhammer",
		Short:   "StableNet stress testing tool",
		Long:    `TxHammer is a CLI tool for stress testing StableNet L1 blockchain networks.`,
		Version: version,
		RunE:    run,
	}

	// Register flags
	registerFlags(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func registerFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	// Required flags
	flags.StringVar(&cfg.URL, "url", "", "RPC endpoint URL (required)")
	flags.StringVar(&cfg.PrivateKey, "private-key", "", "Master account private key (hex)")
	flags.StringVar(&cfg.Mnemonic, "mnemonic", "", "BIP39 mnemonic (alternative to private-key)")

	// Test configuration
	flags.StringVar(&cfg.Mode, "mode", "TRANSFER", "Test mode: TRANSFER, FEE_DELEGATION, CONTRACT_DEPLOY, CONTRACT_CALL, ERC20_TRANSFER, LONG_SENDER, ANALYZE_BLOCKS, ERC721_MINT")
	flags.Uint64Var(&cfg.SubAccounts, "sub-accounts", 10, "Number of sub-accounts")
	flags.Uint64Var(&cfg.Transactions, "transactions", 100, "Total number of transactions")
	flags.Uint64Var(&cfg.BatchSize, "batch", 100, "Batch size for JSON-RPC requests")

	// Chain configuration
	flags.Uint64Var(&cfg.ChainID, "chain-id", 0, "Chain ID (auto-detect if not specified)")
	flags.Uint64Var(&cfg.GasLimit, "gas-limit", 21000, "Gas limit per transaction")
	flags.StringVar(&cfg.GasPrice, "gas-price", "", "Gas price (auto if not specified)")

	// Fee Delegation mode
	flags.StringVar(&cfg.FeePayerKey, "fee-payer-key", "", "Fee payer private key for FEE_DELEGATION mode")

	// Contract mode
	flags.StringVar(&cfg.Contract, "contract", "", "Target contract address")
	flags.StringVar(&cfg.Method, "method", "", "Contract method signature")
	flags.StringVar(&cfg.Args, "args", "", "Method arguments (JSON array)")

	// Output
	flags.StringVar(&cfg.Output, "output", "", "Output JSON file path")
	flags.BoolVar(&cfg.Verbose, "verbose", false, "Enable verbose logging")

	// Advanced
	flags.DurationVar(&cfg.Timeout, "timeout", 0, "Timeout duration (default: 5m)")
	flags.Uint64Var(&cfg.RateLimit, "rate-limit", 0, "Max transactions per second (0 = unlimited)")

	// Run configuration flags
	flags.BoolVar(&runCfg.SkipDistribution, "skip-distribution", false, "Skip fund distribution (assume accounts are funded)")
	flags.BoolVar(&runCfg.SkipCollection, "skip-collection", false, "Skip receipt collection (fire-and-forget mode)")
	flags.BoolVar(&runCfg.ExportReport, "export", true, "Export report to files")
	flags.StringVar(&runCfg.OutputDir, "output-dir", "./reports", "Output directory for reports")
	flags.BoolVar(&runCfg.StreamingMode, "streaming", false, "Use streaming mode instead of batch mode")
	flags.Float64Var(&runCfg.StreamingRate, "streaming-rate", 1000, "Rate limit for streaming mode (tx/s)")
	flags.BoolVar(&runCfg.DryRun, "dry-run", false, "Build transactions but don't send them")

	// Prometheus metrics flags
	flags.BoolVar(&cfg.MetricsEnabled, "metrics", false, "Enable Prometheus metrics endpoint")
	flags.IntVar(&cfg.MetricsPort, "metrics-port", 9090, "Port for Prometheus metrics endpoint")

	// Long Sender mode flags
	flags.DurationVar(&cfg.Duration, "duration", 0, "Test duration for LONG_SENDER mode (e.g., 5m, 1h, 24h)")
	flags.Float64Var(&cfg.TargetTPS, "tps", 100, "Target TPS for LONG_SENDER mode")
	flags.IntVar(&cfg.Workers, "workers", 10, "Number of concurrent workers for LONG_SENDER mode")

	// Block Analyzer mode flags
	flags.Int64Var(&cfg.BlockStart, "block-start", 0, "Start block number for ANALYZE_BLOCKS mode")
	flags.Int64Var(&cfg.BlockEnd, "block-end", 0, "End block number for ANALYZE_BLOCKS mode")
	flags.Int64Var(&cfg.BlockRange, "block-range", 100, "Number of recent blocks to analyze for ANALYZE_BLOCKS mode")

	// ERC721 Mint mode flags
	flags.StringVar(&cfg.NFTName, "nft-name", "TxHammerNFT", "NFT collection name for ERC721_MINT mode")
	flags.StringVar(&cfg.NFTSymbol, "nft-symbol", "TXHNFT", "NFT collection symbol for ERC721_MINT mode")
	flags.StringVar(&cfg.TokenURI, "token-uri", "https://txhammer.io/nft/", "Base token URI for ERC721_MINT mode")

	// Mark required flags
	_ = cmd.MarkFlagRequired("url")
}

func run(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// Create and run pipeline
	p, err := pipeline.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create pipeline: %w", err)
	}
	defer p.Close()

	// Apply run configuration
	p.WithRunConfig(runCfg)

	// Execute pipeline
	result, err := p.Execute(ctx)
	if err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}

	// Exit with error if pipeline failed
	if !result.Success() {
		return fmt.Errorf("stress test completed with errors")
	}

	return nil
}
