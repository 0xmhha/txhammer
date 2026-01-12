# TxHammer

A stress testing CLI tool for StableNet L1 blockchain.

## Overview

TxHammer is a stress testing tool designed to measure the performance of StableNet (EVM-compatible PoA blockchain) networks. It generates and sends large volumes of transactions to measure performance metrics such as TPS, gas utilization, and latency.

## Features

- **Multiple Transaction Types**
  - EIP-1559 dynamic fee transactions (Type 0x02)
  - Fee Delegation transactions (Type 0x16) - StableNet specific
  - Smart contract deployment/calls
  - ERC20 token transfers

- **High-Performance Send Engine**
  - Efficient bulk sending via JSON-RPC batch requests
  - Streaming mode with rate limiting
  - Concurrency control and retry logic

- **Comprehensive Metrics Collection**
  - TPS (sent/confirmed)
  - Latency distribution (avg, min, max, P50, P95, P99)
  - Gas usage and costs
  - Block-level statistics

- **Multiple Output Formats**
  - Real-time console output
  - JSON reports
  - CSV files (summary, transactions, blocks)

## Installation

### Requirements

- Go 1.21 or higher
- Access to a StableNet node (HTTP/WebSocket RPC)

### Build

```bash
make build
```

The binary will be created at `build/txhammer`.

## Quick Start

### Basic Transfer Test

The simplest form of stress test. Distributes funds from the master account to sub-accounts, then each sub-account sends transfer transactions to itself.

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --mode TRANSFER \
  --sub-accounts 10 \
  --transactions 1000 \
  --batch 100
```

### Fee Delegation Test (StableNet Only)

Tests StableNet's Fee Delegation (Type 0x16) feature where a fee payer pays gas costs on behalf of users.

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xSENDER_KEY \
  --fee-payer-key 0xFEE_PAYER_KEY \
  --mode FEE_DELEGATION \
  --sub-accounts 5 \
  --transactions 500
```

### ERC20 Token Transfer Test

Tests calling the transfer function of an ERC20 token contract.

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --mode ERC20_TRANSFER \
  --contract 0xTOKEN_CONTRACT_ADDRESS \
  --sub-accounts 10 \
  --transactions 500 \
  --gas-limit 65000
```

### Smart Contract Deployment Test

Tests network performance by repeatedly deploying smart contracts.

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --mode CONTRACT_DEPLOY \
  --sub-accounts 5 \
  --transactions 100 \
  --gas-limit 200000
```

### Contract Method Call Test

Repeatedly calls a method on a specific contract.

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --mode CONTRACT_CALL \
  --contract 0xCONTRACT_ADDRESS \
  --method "setValue(uint256)" \
  --sub-accounts 10 \
  --transactions 1000 \
  --gas-limit 100000
```

## Advanced Usage

### Streaming Mode

Uses streaming mode with rate limiting instead of batch sending. Suitable for sustained load testing.

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --streaming \
  --streaming-rate 100 \
  --transactions 10000
```

### Dry Run Mode

Builds transactions without actually sending them. Useful for configuration validation.

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --dry-run \
  --transactions 1000
```

### Skip Fund Distribution

If sub-accounts already have sufficient funds, you can skip the distribution stage.

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --skip-distribution \
  --transactions 1000
```

### Fire-and-Forget Mode

Sends transactions without collecting results. Useful for testing maximum send throughput.

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --skip-collection \
  --transactions 10000
```

### Custom Report Directory

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --export \
  --output-dir ./my-reports \
  --transactions 1000
```

## Command Line Flags

### Required Settings

| Flag | Description |
|------|-------------|
| `--url` | RPC endpoint URL (http:// or ws://) |
| `--private-key` | Master account private key (0x prefix + 64 hex chars) |
| `--mnemonic` | BIP39 mnemonic (alternative to private-key) |

### Test Settings

| Flag | Default | Description |
|------|---------|-------------|
| `--mode` | `TRANSFER` | Test mode |
| `--sub-accounts` | `10` | Number of sub-accounts |
| `--transactions` | `100` | Total number of transactions |
| `--batch` | `100` | JSON-RPC batch size |

### Chain Settings

| Flag | Default | Description |
|------|---------|-------------|
| `--chain-id` | (auto) | Chain ID (auto-detected if not specified) |
| `--gas-limit` | `21000` | Gas limit per transaction |
| `--gas-price` | (auto) | Gas price (auto-detected if not specified) |

### Mode-Specific Settings

| Flag | Description |
|------|-------------|
| `--fee-payer-key` | Fee Delegation mode: Fee payer's private key |
| `--contract` | Contract/ERC20 mode: Target contract address |
| `--method` | Contract Call mode: Method signature |
| `--args` | Contract Call mode: Method arguments (JSON array) |

### Execution Options

| Flag | Default | Description |
|------|---------|-------------|
| `--skip-distribution` | `false` | Skip fund distribution stage |
| `--skip-collection` | `false` | Skip receipt collection stage |
| `--streaming` | `false` | Use streaming mode |
| `--streaming-rate` | `1000` | Streaming rate (tx/s) |
| `--dry-run` | `false` | Build only, don't send |

### Output Settings

| Flag | Default | Description |
|------|---------|-------------|
| `--export` | `true` | Export report files |
| `--output-dir` | `./reports` | Report output directory |
| `--output` | - | Output JSON file path (legacy) |
| `--verbose` | `false` | Enable verbose logging |

### Advanced Settings

| Flag | Default | Description |
|------|---------|-------------|
| `--timeout` | `5m` | Overall timeout |
| `--rate-limit` | `0` | Max transactions per second (0=unlimited) |

## Test Modes

| Mode | Gas Limit | Description |
|------|-----------|-------------|
| `TRANSFER` | 21000 | Simple native coin transfer (self-transfer) |
| `FEE_DELEGATION` | 21000 | Fee delegated transactions (StableNet Type 0x16) |
| `CONTRACT_DEPLOY` | 200000 | SimpleStorage contract deployment |
| `CONTRACT_CALL` | 100000 | Call specified contract method |
| `ERC20_TRANSFER` | 65000 | ERC20 token transfer |

## Output & Reports

### Report Files

When `--export` is enabled, the following files are generated:

```
reports/
├── report_20240115_143052.json      # Full metrics (JSON)
├── summary_20240115_143052.csv      # Summary metrics
├── transactions_20240115_143052.csv # Per-transaction details
└── blocks_20240115_143052.csv       # Per-block statistics
```

### JSON Report Structure

```json
{
  "test_name": "stress-test",
  "start_time": "2024-01-15T14:30:52+09:00",
  "end_time": "2024-01-15T14:31:07+09:00",
  "duration": "15.234s",
  "summary": {
    "total_sent": 1000,
    "total_confirmed": 998,
    "total_failed": 2,
    "success_rate": 99.8,
    "tps": 65.64,
    "confirmed_tps": 65.51
  },
  "latency": {
    "average": "234ms",
    "min": "45ms",
    "max": "1.2s",
    "p50": "198ms",
    "p95": "456ms",
    "p99": "890ms"
  },
  "gas": {
    "total_used": 20958000,
    "average_used": 21000,
    "total_cost": "20958000000000000"
  }
}
```

## Troubleshooting

### "insufficient funds" Error

Ensure the master account has sufficient balance. Required funds are calculated as:

```
Required = (gas_limit × gas_price × txs_per_account × 1.2) × sub_accounts
         + (21000 × gas_price × sub_accounts)  // distribution tx gas
```

### "nonce too low" Error

Previous test transactions may still be processing. Wait a moment and try again, or run with `--skip-distribution`.

### Fee Delegation Errors

- Verify `--fee-payer-key` format is correct (0x + 64 hex chars)
- Ensure fee payer account has sufficient balance
- Confirm the node supports Type 0x16 transactions

### Low TPS

- Increase `--batch` size (e.g., 200, 500)
- Increase `--sub-accounts` to increase parallelism
- Check network latency to the node
- Verify node's processing capacity

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - Internal architecture and component details
- [Development Guide](docs/DEVELOPMENT.md) - Development and contribution instructions
- [Fee Delegation Spec](docs/FEE_DELEGATION.md) - Technical specification for Type 0x16 transactions

## License

Apache-2.0
