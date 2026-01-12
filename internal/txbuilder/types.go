package txbuilder

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TxType represents the transaction type
type TxType byte

const (
	// Standard Ethereum transaction types
	TxTypeLegacy     TxType = 0x00
	TxTypeAccessList TxType = 0x01
	TxTypeDynamicFee TxType = 0x02
	TxTypeBlob       TxType = 0x03

	// StableNet specific transaction types
	TxTypeFeeDelegation TxType = 0x16
)

// TxRequest represents a transaction request
type TxRequest struct {
	From     common.Address
	To       *common.Address
	Value    *big.Int
	Data     []byte
	Nonce    uint64
	Gas      uint64
	GasPrice *big.Int // for legacy tx
	GasTipCap *big.Int // for EIP-1559
	GasFeeCap *big.Int // for EIP-1559
	ChainID  *big.Int
}

// FeeDelegationRequest extends TxRequest for fee delegation
type FeeDelegationRequest struct {
	TxRequest
	FeePayer    common.Address
	FeePayerKey *ecdsa.PrivateKey
}

// SignedTx represents a signed transaction ready to send
type SignedTx struct {
	Tx       *types.Transaction
	RawTx    []byte
	Hash     common.Hash
	From     common.Address
	Nonce    uint64
	GasLimit uint64
}

// FeeDelegationTx represents a fee delegation transaction (Type 0x16)
// This is StableNet-specific transaction type
type FeeDelegationTx struct {
	// Sender transaction (EIP-1559 style)
	ChainID    *big.Int
	Nonce      uint64
	GasTipCap  *big.Int
	GasFeeCap  *big.Int
	Gas        uint64
	To         *common.Address
	Value      *big.Int
	Data       []byte
	AccessList types.AccessList

	// Sender signature
	V *big.Int
	R *big.Int
	S *big.Int

	// Fee payer
	FeePayer *common.Address

	// Fee payer signature
	FV *big.Int
	FR *big.Int
	FS *big.Int
}

// BuilderConfig holds configuration for transaction building
type BuilderConfig struct {
	ChainID   *big.Int
	GasLimit  uint64
	GasPrice  *big.Int
	GasTipCap *big.Int
	GasFeeCap *big.Int
}

// ContractCallRequest represents a contract call request
type ContractCallRequest struct {
	TxRequest
	Method string
	Args   []interface{}
}

// ERC20TransferRequest represents an ERC20 transfer request
type ERC20TransferRequest struct {
	TxRequest
	TokenAddress common.Address
	Recipient    common.Address
	Amount       *big.Int
}
