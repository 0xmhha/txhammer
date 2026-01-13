# TxHammer

![Go Version](https://img.shields.io/badge/Go-1.24%20%7C%20go1.24.11-00ADD8?logo=go)
![Consensus](https://img.shields.io/badge/Consensus-QBFT--based%20WBFT-4c1)
![License](https://img.shields.io/badge/License-Apache%202.0-blue)

A stress testing CLI tool for StableNet L1 blockchain.

## Overview

TxHammer is a stress testing tool designed to measure the performance of StableNet (EVM-compatible blockchain that uses a QBFT-based WBFT consensus) networks. It generates and sends large volumes of transactions to measure performance metrics such as TPS, gas utilization, and latency.

## Features

- **Multiple Transaction Types**
  - EIP-1559 dynamic fee transactions (Type 0x02)
  - Fee Delegation transactions (Type 0x16) - StableNet specific
  - Smart contract deployment/calls
  - ERC20 token transfers
  - ERC721 NFT minting

- **High-Performance Send Engine**
  - Efficient bulk sending via JSON-RPC batch requests
  - Streaming mode with rate limiting
  - Concurrency control and retry logic
  - **Long Sender mode** for duration-based continuous testing

- **Comprehensive Metrics Collection**
  - TPS (sent/confirmed)
  - Latency distribution (avg, min, max, P50, P95, P99)
  - Gas usage and costs
  - Block-level statistics
  - **Prometheus metrics endpoint** for monitoring integration

- **Block Analysis**
  - Analyze existing blocks without sending transactions
  - Calculate historical TPS from block data
  - Export analysis results to CSV

- **Real-time Monitoring**
  - Live TPS display with rolling window calculation
  - Prometheus metrics for Grafana integration

- **Multiple Output Formats**
  - Real-time console output
  - JSON reports
  - CSV files (summary, transactions, blocks)

## Installation

### Requirements

- Go 1.24 (tested with toolchain `go1.24.11` as defined in `go.mod`)
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

### ERC721 NFT Minting Test

Tests NFT minting performance. Automatically deploys an NFT contract if no contract address is specified.

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --mode ERC721_MINT \
  --nft-name "TestNFT" \
  --nft-symbol "TNFT" \
  --token-uri "https://example.com/nft/" \
  --sub-accounts 5 \
  --transactions 500 \
  --gas-limit 150000
```

### Long Sender Mode (Duration-Based Testing)

Continuously sends transactions for a specified duration at a target TPS rate. Ideal for sustained load testing.

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --mode LONG_SENDER \
  --duration 10m \
  --tps 500 \
  --workers 20 \
  --sub-accounts 10
```

### Block Analyzer Mode

Analyzes existing blocks without sending transactions. Useful for measuring historical network performance.

```bash
# Analyze the last 100 blocks
./build/txhammer \
  --url http://localhost:8545 \
  --mode ANALYZE_BLOCKS \
  --block-range 100

# Analyze a specific block range
./build/txhammer \
  --url http://localhost:8545 \
  --mode ANALYZE_BLOCKS \
  --block-start 1000 \
  --block-end 2000
```

## Advanced Usage

### Custom Transfer Value

By default, each transfer sends 1 wei. You can customize the transfer value:

```bash
# Transfer 0.001 ETH per transaction
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --value 1000000000000000 \
  --transactions 100

# Transfer 0 wei (gas cost only)
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --value 0 \
  --transactions 100
```

**Common value units:**
| Amount | Wei Value |
|--------|-----------|
| 1 wei | `1` |
| 1 Gwei | `1000000000` |
| 0.001 ETH | `1000000000000000` |
| 0.01 ETH | `10000000000000000` |
| 0.1 ETH | `100000000000000000` |
| 1 ETH | `1000000000000000000` |

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

### Prometheus Metrics

Enable Prometheus metrics endpoint for integration with monitoring systems like Grafana.

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xYOUR_PRIVATE_KEY \
  --metrics \
  --metrics-port 9090 \
  --mode LONG_SENDER \
  --duration 1h \
  --tps 100
```

Access metrics at `http://localhost:9090/metrics`. Available metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `txhammer_tx_sent_total` | Counter | Total transactions sent |
| `txhammer_tx_confirmed_total` | Counter | Total transactions confirmed |
| `txhammer_tx_failed_total` | Counter | Total transactions failed |
| `txhammer_tx_latency_seconds` | Histogram | Transaction latency distribution |
| `txhammer_current_tps` | Gauge | Current TPS (rolling window) |
| `txhammer_confirmed_tps` | Gauge | Confirmed TPS |
| `txhammer_pending_tx_count` | Gauge | Pending transaction count |
| `txhammer_gas_used_total` | Counter | Total gas used |
| `txhammer_stage_duration_seconds` | Histogram | Pipeline stage durations |

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
| `--value` | `1` | Transfer value in wei (default: 1 wei) |

### Mode-Specific Settings

| Flag | Description |
|------|-------------|
| `--fee-payer-key` | Fee Delegation mode: Fee payer's private key |
| `--contract` | Contract/ERC20/ERC721 mode: Target contract address |
| `--method` | Contract Call mode: Method signature |
| `--args` | Contract Call mode: Method arguments (JSON array) |

### Long Sender Mode Settings

| Flag | Default | Description |
|------|---------|-------------|
| `--duration` | - | Test duration (e.g., `5m`, `1h`, `24h`) |
| `--tps` | `100` | Target transactions per second |
| `--workers` | `10` | Number of concurrent workers |

### Block Analyzer Mode Settings

| Flag | Default | Description |
|------|---------|-------------|
| `--block-start` | `0` | Start block number |
| `--block-end` | `0` | End block number (0 = latest) |
| `--block-range` | `100` | Number of recent blocks to analyze |

### ERC721 Mint Mode Settings

| Flag | Default | Description |
|------|---------|-------------|
| `--nft-name` | `TxHammerNFT` | NFT collection name |
| `--nft-symbol` | `TXHNFT` | NFT collection symbol |
| `--token-uri` | `https://txhammer.io/nft/` | Base token URI |

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

### Monitoring Settings

| Flag | Default | Description |
|------|---------|-------------|
| `--metrics` | `false` | Enable Prometheus metrics endpoint |
| `--metrics-port` | `9090` | Prometheus metrics port |

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
| `ERC721_MINT` | 150000 | ERC721 NFT minting |
| `LONG_SENDER` | 21000 | Duration-based continuous sending (requires `--duration`) |
| `ANALYZE_BLOCKS` | - | Block analysis only (no transactions sent) |

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
Required = ((gas_limit × gas_price + value) × txs_per_account × 1.2) × sub_accounts
         + (21000 × gas_price × sub_accounts)  // distribution tx gas
```

Note: The `--value` flag affects the required funds. Higher transfer values require more balance per sub-account.

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

This project is licensed under the [Apache License 2.0](LICENSE).
