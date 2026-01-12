# Fee Delegation (Type 0x16) Technical Specification

This document describes the technical details of StableNet's Fee Delegation transaction type.

## Overview

Fee Delegation allows a fee payer to pay gas costs on behalf of the transaction sender. This is a StableNet-specific feature using transaction type `0x16`.

## Transaction Structure

```
Type 0x16 Transaction:
0x16 || RLP([
  // Sender Transaction (EIP-1559 style)
  chainId,
  nonce,
  maxPriorityFeePerGas,
  maxFeePerGas,
  gasLimit,
  to,
  value,
  data,
  accessList,
  // Sender Signature
  v, r, s,
  // Fee Payer
  feePayer,
  // Fee Payer Signature
  fv, fr, fs
])
```

## Signing Process

The signing process involves two parties: the sender and the fee payer.

### Step 1: Sender Creates Transaction Hash

The sender creates an EIP-1559 style transaction hash:

```
senderTxHash = keccak256(0x02 || RLP([
  chainId,
  nonce,
  maxPriorityFeePerGas,
  maxFeePerGas,
  gasLimit,
  to,
  value,
  data,
  accessList
]))
```

### Step 2: Sender Signs

The sender signs the transaction hash to produce `(v, r, s)`:

```
senderSig = sign(senderPrivateKey, senderTxHash)
```

### Step 3: Fee Payer Creates Hash

The fee payer creates a hash that includes the sender's transaction and signature:

```
feePayerHash = keccak256(0x16 || RLP([
  chainId,
  nonce,
  maxPriorityFeePerGas,
  maxFeePerGas,
  gasLimit,
  to,
  value,
  data,
  accessList,
  v, r, s,
  feePayer
]))
```

### Step 4: Fee Payer Signs

The fee payer signs their hash to produce `(fv, fr, fs)`:

```
feePayerSig = sign(feePayerPrivateKey, feePayerHash)
```

### Step 5: Final Encoding

The complete transaction is RLP encoded with the `0x16` type prefix.

## Implementation Notes

### Gas Estimation

When estimating gas for Fee Delegation transactions, the RPC node must support the `0x16` transaction type. The gas cost is similar to a standard EIP-1559 transaction.

### Nonce Management

- The **sender's nonce** is used for transaction ordering
- The **fee payer's nonce** is NOT used in the transaction
- The fee payer's balance is used to pay gas costs

### Error Handling

Common errors when using Fee Delegation:

| Error | Cause | Solution |
|-------|-------|----------|
| `invalid sender signature` | Sender signature doesn't match transaction | Verify sender private key and transaction data |
| `invalid fee payer signature` | Fee payer signature doesn't match | Verify fee payer private key |
| `insufficient funds for gas` | Fee payer balance too low | Ensure fee payer has sufficient balance |
| `unknown transaction type` | Node doesn't support Type 0x16 | Use a StableNet-compatible node |

### Balance Requirements

For Fee Delegation transactions:

- **Sender**: Needs balance for `value` only (the amount being transferred)
- **Fee Payer**: Needs balance for `gasLimit × maxFeePerGas`

## Code Example

```go
// Create Fee Delegation transaction
tx := &FeeDelegationTx{
    ChainID:              chainID,
    Nonce:                senderNonce,
    MaxPriorityFeePerGas: maxPriorityFee,
    MaxFeePerGas:         maxFee,
    GasLimit:             gasLimit,
    To:                   &recipient,
    Value:                value,
    Data:                 nil,
    AccessList:           nil,
}

// Sign as sender
senderSig, err := SignAsSender(senderKey, tx)

// Sign as fee payer
feePayerSig, err := SignAsFeePayer(feePayerKey, tx, senderSig)

// Encode final transaction
rawTx, err := EncodeFeeDelegationTx(tx, senderSig, feePayerSig)
```

## Testing Fee Delegation

Use the `FEE_DELEGATION` mode in TxHammer:

```bash
./build/txhammer \
  --url http://localhost:8545 \
  --private-key 0xSENDER_KEY \
  --fee-payer-key 0xFEE_PAYER_KEY \
  --mode FEE_DELEGATION \
  --transactions 100
```

This will:
1. Create transactions signed by the sender
2. Have the fee payer sign to pay gas costs
3. Submit the transactions to the network
4. Collect metrics on transaction performance
