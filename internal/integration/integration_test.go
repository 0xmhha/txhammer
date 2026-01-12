// Package integration provides integration tests for txhammer.
// These tests require a running StableNet node (or EVM-compatible node) and should be run with:
//
//	go test -tags=integration ./internal/integration/...
//
// Environment variables:
//   - RPC_URL: RPC endpoint URL (default: http://localhost:8545)
//   - PRIVATE_KEY: Private key with funds for testing
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/piatoss3612/txhammer/internal/client"
	"github.com/piatoss3612/txhammer/internal/config"
	"github.com/piatoss3612/txhammer/internal/pipeline"
	"github.com/piatoss3612/txhammer/internal/wallet"
)

const (
	defaultRPCURL = "http://localhost:8545"
)

// skipIfNoRPC skips the test if no RPC endpoint is available
func skipIfNoRPC(t *testing.T) {
	t.Helper()
	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		rpcURL = defaultRPCURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli, err := client.New(rpcURL)
	if err != nil {
		t.Skipf("Skipping integration test: cannot connect to RPC at %s: %v", rpcURL, err)
	}
	defer cli.Close()

	_, err = cli.ChainID(ctx)
	if err != nil {
		t.Skipf("Skipping integration test: RPC not responding at %s: %v", rpcURL, err)
	}
}

// skipIfNoPrivateKey skips the test if no private key is provided
func skipIfNoPrivateKey(t *testing.T) {
	t.Helper()
	if os.Getenv("PRIVATE_KEY") == "" {
		t.Skip("Skipping integration test: PRIVATE_KEY environment variable not set")
	}
}

// getTestConfig returns a test configuration from environment variables
func getTestConfig(t *testing.T) *config.Config {
	t.Helper()

	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		rpcURL = defaultRPCURL
	}

	privateKey := os.Getenv("PRIVATE_KEY")
	if privateKey == "" {
		t.Fatal("PRIVATE_KEY environment variable required")
	}

	return &config.Config{
		URL:          rpcURL,
		PrivateKey:   privateKey,
		Mode:         "TRANSFER",
		SubAccounts:  2,
		Transactions: 10,
		BatchSize:    5,
		GasLimit:     21000,
		Timeout:      2 * time.Minute,
	}
}

// TestClientConnection tests basic RPC connectivity
func TestClientConnection(t *testing.T) {
	skipIfNoRPC(t)

	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		rpcURL = defaultRPCURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli, err := client.New(rpcURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer cli.Close()

	// Test ChainID
	chainID, err := cli.ChainID(ctx)
	if err != nil {
		t.Fatalf("Failed to get chain ID: %v", err)
	}
	t.Logf("Chain ID: %s", chainID.String())

	// Test BlockNumber
	blockNum, err := cli.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("Failed to get block number: %v", err)
	}
	t.Logf("Block Number: %d", blockNum)

	// Test GasPrice
	gasPrice, err := cli.SuggestGasPrice(ctx)
	if err != nil {
		t.Fatalf("Failed to get gas price: %v", err)
	}
	t.Logf("Gas Price: %s wei", gasPrice.String())
}

// TestWalletCreation tests wallet creation from private key
func TestWalletCreation(t *testing.T) {
	skipIfNoRPC(t)
	skipIfNoPrivateKey(t)

	privateKey := os.Getenv("PRIVATE_KEY")

	w, err := wallet.NewFromPrivateKey(privateKey, 5)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	t.Logf("Master Address: %s", w.MasterAddress().Hex())
	t.Logf("Sub Accounts: %d", len(w.SubAddresses()))

	for i, addr := range w.SubAddresses() {
		t.Logf("  Sub Account %d: %s", i, addr.Hex())
	}
}

// TestWalletBalance tests checking wallet balance
func TestWalletBalance(t *testing.T) {
	skipIfNoRPC(t)
	skipIfNoPrivateKey(t)

	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		rpcURL = defaultRPCURL
	}
	privateKey := os.Getenv("PRIVATE_KEY")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli, err := client.New(rpcURL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer cli.Close()

	w, err := wallet.NewFromPrivateKey(privateKey, 1)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	balance, err := cli.BalanceAt(ctx, w.MasterAddress(), nil)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}

	t.Logf("Master Balance: %s wei", balance.String())

	if balance.Sign() == 0 {
		t.Log("Warning: Master account has zero balance")
	}
}

// TestPipelineCreation tests pipeline creation
func TestPipelineCreation(t *testing.T) {
	skipIfNoRPC(t)
	skipIfNoPrivateKey(t)

	cfg := getTestConfig(t)

	p, err := pipeline.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer p.Close()

	t.Log("Pipeline created successfully")
}

// TestPipelineDryRun tests pipeline dry run (builds but doesn't send)
func TestPipelineDryRun(t *testing.T) {
	skipIfNoRPC(t)
	skipIfNoPrivateKey(t)

	cfg := getTestConfig(t)
	cfg.Transactions = 5 // Small number for testing

	p, err := pipeline.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer p.Close()

	runCfg := &pipeline.RunConfig{
		SkipDistribution: true, // Skip distribution for dry run
		SkipCollection:   true,
		ExportReport:     false,
		OutputDir:        t.TempDir(),
		DryRun:           true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	result, err := p.WithRunConfig(runCfg).Execute(ctx)
	if err != nil {
		t.Fatalf("Pipeline dry run failed: %v", err)
	}

	t.Logf("Dry run completed in %s", result.Duration)
	t.Logf("Stages completed: %d", len(result.StageResults))

	for _, sr := range result.StageResults {
		status := "✓"
		if !sr.Success {
			status = "✗"
		}
		t.Logf("  %s %s: %s", status, sr.Stage.String(), sr.Duration)
	}
}

// TestFullPipeline runs the complete pipeline (requires funded account)
func TestFullPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full pipeline test in short mode")
	}

	skipIfNoRPC(t)
	skipIfNoPrivateKey(t)

	cfg := getTestConfig(t)
	cfg.Transactions = 5      // Small number for testing
	cfg.SubAccounts = 2       // Minimal sub-accounts
	cfg.BatchSize = 5         // Small batches
	cfg.GasLimit = 21000      // Standard transfer gas
	cfg.Timeout = time.Minute // Short timeout

	p, err := pipeline.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer p.Close()

	runCfg := &pipeline.RunConfig{
		SkipDistribution: false,
		SkipCollection:   false,
		ExportReport:     true,
		OutputDir:        t.TempDir(),
		DryRun:           false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	result, err := p.WithRunConfig(runCfg).Execute(ctx)
	if err != nil {
		t.Fatalf("Pipeline execution failed: %v", err)
	}

	// Check results
	if !result.Success() {
		t.Errorf("Pipeline did not complete successfully")
		for _, e := range result.Errors {
			t.Errorf("  Error: %v", e)
		}
	}

	t.Logf("Full pipeline completed in %s", result.Duration)
	t.Logf("Total Transactions: %d", result.TotalTransactions)
	t.Logf("TPS: %.2f", result.TPS)
}
