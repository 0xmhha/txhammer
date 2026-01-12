# Architecture

This document describes the internal architecture of TxHammer.

## Project Structure

```
txhammer/
├── cmd/
│   └── main.go              # CLI entry point, Cobra command definitions
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration struct and validation
│   ├── client/
│   │   └── client.go        # RPC client, batch requests
│   ├── wallet/
│   │   └── wallet.go        # HD Wallet, key management
│   ├── txbuilder/
│   │   ├── types.go         # Transaction type definitions
│   │   ├── builder.go       # Builder interface
│   │   ├── transfer.go      # EIP-1559 transfer builder
│   │   ├── fee_delegation.go # Fee Delegation (Type 0x16) builder
│   │   ├── contract.go      # Contract deploy/call builder
│   │   ├── erc20.go         # ERC20 transfer builder
│   │   └── factory.go       # Builder factory
│   ├── distributor/
│   │   ├── types.go         # Distribution type definitions
│   │   └── distributor.go   # Fund distribution logic
│   ├── batcher/
│   │   ├── types.go         # Batch type definitions
│   │   ├── batcher.go       # Batch send logic
│   │   └── streamer.go      # Streaming send logic
│   ├── collector/
│   │   ├── types.go         # Metrics type definitions
│   │   ├── collector.go     # Receipt collection logic
│   │   └── exporter.go      # JSON/CSV export
│   └── pipeline/
│       ├── types.go         # Pipeline type definitions
│       └── pipeline.go      # Main orchestration
├── pkg/
│   └── types/
│       └── types.go         # Public types (legacy)
├── Makefile
├── go.mod
└── README.md
```

## Pipeline Stages

TxHammer operates through a 6-stage pipeline:

```
┌─────────────────────────────────────────────────────────────────┐
│                        TxHammer Pipeline                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Stage 1: INITIALIZE                                            │
│  ├─ Query and validate Chain ID                                 │
│  ├─ Initialize components (distributor, builder, batcher,       │
│  │   collector)                                                 │
│  └─ Check master account balance                                │
│                                                                  │
│  Stage 2: DISTRIBUTE (can be skipped with --skip-distribution)  │
│  ├─ Check sub-account balances                                  │
│  ├─ Calculate required funds (gas × txs × price × 1.2)         │
│  ├─ Distribute funds via EIP-1559 transfer transactions         │
│  └─ Wait for distribution transaction confirmations             │
│                                                                  │
│  Stage 3: BUILD                                                  │
│  ├─ Create Builder based on mode                                │
│  ├─ Query nonce for each sub-account                            │
│  └─ Generate signed transaction batches                         │
│                                                                  │
│  Stage 4: SEND (skipped with --dry-run)                        │
│  ├─ Batch mode: Mass send via JSON-RPC batch requests          │
│  └─ Streaming mode: Sequential send with rate limiting          │
│                                                                  │
│  Stage 5: COLLECT (can be skipped with --skip-collection)       │
│  ├─ Poll for transaction receipts                               │
│  ├─ Collect block-level metrics                                 │
│  └─ Calculate latency and gas statistics                        │
│                                                                  │
│  Stage 6: REPORT                                                 │
│  ├─ Calculate final metrics                                     │
│  └─ Export JSON/CSV files                                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

### Config (`internal/config`)

- Parses and validates CLI flags
- Provides default values for optional settings
- Validates configuration consistency

### Client (`internal/client`)

- Wraps go-ethereum's ethclient
- Provides JSON-RPC batch request support
- Handles connection management

### Wallet (`internal/wallet`)

- Manages HD wallet derivation (BIP39/BIP44)
- Supports both mnemonic and raw private key initialization
- Derives deterministic sub-accounts from master key

### TxBuilder (`internal/txbuilder`)

- Factory pattern for creating mode-specific builders
- Each builder creates signed transactions for its mode
- Supports gas estimation and nonce management

### Distributor (`internal/distributor`)

- Calculates required funds for sub-accounts
- Distributes funds from master account
- Tracks distribution transaction confirmations

### Batcher (`internal/batcher`)

- Splits transactions into batches
- Sends batches via JSON-RPC batch requests
- Supports streaming mode with rate limiting

### Collector (`internal/collector`)

- Polls for transaction receipts
- Aggregates metrics (latency, gas, block stats)
- Calculates percentiles (P50, P95, P99)

### Pipeline (`internal/pipeline`)

- Orchestrates the 6-stage execution flow
- Manages stage transitions and error handling
- Coordinates between all components
