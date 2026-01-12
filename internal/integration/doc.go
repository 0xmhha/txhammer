// Package integration provides integration tests for txhammer.
//
// These tests verify the complete workflow of the stress testing tool
// against a real StableNet node. They are designed to be skipped when
// no node is available, making them safe to include in CI/CD pipelines.
//
// # Running Integration Tests
//
// Basic integration tests (connection, wallet creation):
//
//	RPC_URL=http://localhost:8545 go test ./internal/integration/...
//
// Full integration tests (requires funded account):
//
//	RPC_URL=http://localhost:8545 \
//	PRIVATE_KEY=0x... \
//	go test ./internal/integration/...
//
// Skip integration tests in CI:
//
//	go test -short ./...
//
// # Environment Variables
//
//   - RPC_URL: RPC endpoint URL (default: http://localhost:8545)
//   - PRIVATE_KEY: Private key with funds for testing (hex format, with or without 0x prefix)
//
// # Test Categories
//
// Connection tests verify RPC connectivity and basic chain queries.
// These run without a funded account.
//
// Wallet tests verify wallet creation and address derivation.
// These run without sending transactions.
//
// Pipeline tests verify the complete stress testing workflow.
// These require a funded account to send transactions.
//
// # Local Development
//
// For local testing, you can use anvil (from Foundry):
//
//	# Start a local node
//	anvil
//
//	# Run integration tests with the default anvil private key
//	RPC_URL=http://localhost:8545 \
//	PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
//	go test ./internal/integration/...
package integration
