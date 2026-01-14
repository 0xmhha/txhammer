package client

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Client wraps the Ethereum client with additional functionality
type Client struct {
	eth *ethclient.Client
	rpc *rpc.Client
}

// New creates a new client instance
func New(url string) (*Client, error) {
	rpcClient, err := rpc.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	ethClient := ethclient.NewClient(rpcClient)

	return &Client{
		eth: ethClient,
		rpc: rpcClient,
	}, nil
}

// Close closes the client connection
func (c *Client) Close() {
	c.rpc.Close()
}

// ChainID returns the chain ID
func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
	return c.eth.ChainID(ctx)
}

// BlockNumber returns the latest block number
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	return c.eth.BlockNumber(ctx)
}

// BlockByNumber returns a block by number
func (c *Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return c.eth.BlockByNumber(ctx, number)
}

// BalanceAt returns the balance of an account at a given block
func (c *Client) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return c.eth.BalanceAt(ctx, account, blockNumber)
}

// PendingNonceAt returns the pending nonce for an account
func (c *Client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return c.eth.PendingNonceAt(ctx, account)
}

// SuggestGasPrice returns the suggested gas price
func (c *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return c.eth.SuggestGasPrice(ctx)
}

// SuggestGasTipCap returns the suggested gas tip cap (EIP-1559)
func (c *Client) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return c.eth.SuggestGasTipCap(ctx)
}

// EstimateGas estimates the gas needed for a transaction
func (c *Client) EstimateGas(ctx context.Context, msg *ethereum.CallMsg) (uint64, error) {
	return c.eth.EstimateGas(ctx, *msg)
}

// SendTransaction sends a signed transaction
func (c *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return c.eth.SendTransaction(ctx, tx)
}

// TransactionReceipt returns the receipt of a transaction by hash
func (c *Client) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return c.eth.TransactionReceipt(ctx, txHash)
}

// HeaderByNumber returns the header of a block by number
func (c *Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return c.eth.HeaderByNumber(ctx, number)
}

// BatchCall executes multiple RPC calls in a single request
func (c *Client) BatchCall(b []rpc.BatchElem) error {
	return c.rpc.BatchCall(b)
}

// SendRawTransaction sends a raw transaction via RPC
func (c *Client) SendRawTransaction(ctx context.Context, rawTx []byte) (common.Hash, error) {
	var hash common.Hash
	err := c.rpc.CallContext(ctx, &hash, "eth_sendRawTransaction", "0x"+common.Bytes2Hex(rawTx))
	return hash, err
}

// BatchSendRawTransactions sends multiple raw transactions in a batch
func (c *Client) BatchSendRawTransactions(ctx context.Context, rawTxs [][]byte) ([]common.Hash, error) {
	batch := make([]rpc.BatchElem, len(rawTxs))
	results := make([]common.Hash, len(rawTxs))

	for i, rawTx := range rawTxs {
		batch[i] = rpc.BatchElem{
			Method: "eth_sendRawTransaction",
			Args:   []interface{}{"0x" + common.Bytes2Hex(rawTx)},
			Result: &results[i],
		}
	}

	if err := c.rpc.BatchCallContext(ctx, batch); err != nil {
		return nil, fmt.Errorf("batch call failed: %w", err)
	}

	// Check for individual errors
	for i, elem := range batch {
		if elem.Error != nil {
			return nil, fmt.Errorf("transaction %d failed: %w", i, elem.Error)
		}
	}

	return results, nil
}

// GetBlockGasLimit returns the gas limit of a specific block
func (c *Client) GetBlockGasLimit(ctx context.Context, blockNumber uint64) (uint64, error) {
	block, err := c.eth.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return 0, err
	}
	return block.GasLimit(), nil
}

// GetBlockGasUsed returns the gas used in a specific block
func (c *Client) GetBlockGasUsed(ctx context.Context, blockNumber uint64) (uint64, error) {
	block, err := c.eth.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return 0, err
	}
	return block.GasUsed(), nil
}
