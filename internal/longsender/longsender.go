package longsender

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/time/rate"
)

// LongSender provides duration-based continuous transaction sending
type LongSender struct {
	client  SendClient
	config  *Config
	limiter *rate.Limiter

	// Keys and addresses
	keys      []*ecdsa.PrivateKey
	addresses []common.Address

	// Atomic nonce management per account
	nonces []atomic.Uint64

	// Atomic counters
	sentCount   atomic.Int64
	failedCount atomic.Int64

	// Chain info
	chainID  *big.Int
	gasPrice *big.Int
	gasLimit uint64

	// Callbacks
	callbacks *Callbacks

	// Start time for TPS calculation
	startTime time.Time

	// Error collection
	errors   []error
	errorsMu sync.Mutex
}

// New creates a new LongSender instance
func New(client SendClient, config *Config) *LongSender {
	if config == nil {
		config = DefaultConfig()
	}

	// Create rate limiter
	limiter := rate.NewLimiter(rate.Limit(config.TPS), config.Burst)

	return &LongSender{
		client:   client,
		config:   config,
		limiter:  limiter,
		gasLimit: 21000, // Standard transfer gas limit
		errors:   make([]error, 0),
	}
}

// WithGasLimit sets the gas limit for transactions
func (l *LongSender) WithGasLimit(gasLimit uint64) *LongSender {
	l.gasLimit = gasLimit
	return l
}

// WithGasPrice sets the gas price for transactions
func (l *LongSender) WithGasPrice(gasPrice *big.Int) *LongSender {
	l.gasPrice = gasPrice
	return l
}

// WithCallbacks sets the callbacks for metrics integration
func (l *LongSender) WithCallbacks(callbacks *Callbacks) *LongSender {
	l.callbacks = callbacks
	return l
}

// Run executes the long sender with the given keys and initial nonces
func (l *LongSender) Run(ctx context.Context, keys []*ecdsa.PrivateKey, initialNonces []uint64) (*Result, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys provided")
	}
	if len(keys) != len(initialNonces) {
		return nil, fmt.Errorf("keys and nonces count mismatch")
	}

	// Setup keys and addresses
	l.keys = keys
	l.addresses = make([]common.Address, len(keys))
	l.nonces = make([]atomic.Uint64, len(keys))

	for i, key := range keys {
		l.addresses[i] = crypto.PubkeyToAddress(key.PublicKey)
		l.nonces[i].Store(initialNonces[i])
	}

	// Get chain info
	var err error
	l.chainID, err = l.client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	// Get gas price if not set
	if l.gasPrice == nil {
		l.gasPrice, err = l.client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get gas price: %w", err)
		}
	}

	// Create context with timeout if duration is set
	runCtx := ctx
	var cancel context.CancelFunc
	if l.config.Duration > 0 {
		runCtx, cancel = context.WithTimeout(ctx, l.config.Duration)
		defer cancel()
	}

	l.startTime = time.Now()

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < l.config.Workers; i++ {
		wg.Add(1)
		go l.worker(runCtx, &wg, i)
	}

	// Wait for all workers to finish
	wg.Wait()

	// Calculate results
	duration := time.Since(l.startTime)
	sent := l.sentCount.Load()
	failed := l.failedCount.Load()

	avgTPS := float64(0)
	if duration.Seconds() > 0 {
		avgTPS = float64(sent) / duration.Seconds()
	}

	return &Result{
		TotalSent:     sent,
		TotalFailed:   failed,
		TotalDuration: duration,
		AverageTPS:    avgTPS,
		ActualTPS:     avgTPS,
		Errors:        l.errors,
	}, nil
}

// worker is a goroutine that continuously sends transactions
func (l *LongSender) worker(ctx context.Context, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Wait for rate limiter
			if err := l.limiter.Wait(ctx); err != nil {
				if ctx.Err() != nil {
					return // Context cancelled
				}
				continue
			}

			// Round-robin account selection
			accountIdx := int(l.sentCount.Load()+l.failedCount.Load()) % len(l.keys)

			// Send transaction
			if err := l.sendTransaction(ctx, accountIdx); err != nil {
				l.failedCount.Add(1)
				l.recordError(err)
				if l.callbacks != nil && l.callbacks.OnFailed != nil {
					l.callbacks.OnFailed(err)
				}
			}
		}
	}
}

// sendTransaction creates and sends a single transaction
func (l *LongSender) sendTransaction(ctx context.Context, accountIdx int) error {
	key := l.keys[accountIdx]
	from := l.addresses[accountIdx]
	nonce := l.getNonceAndIncrement(accountIdx)

	// Create transaction (self-transfer)
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   l.chainID,
		Nonce:     nonce,
		GasTipCap: l.gasPrice,
		GasFeeCap: new(big.Int).Mul(l.gasPrice, big.NewInt(2)),
		Gas:       l.gasLimit,
		To:        &from, // Self-transfer
		Value:     big.NewInt(0),
		Data:      nil,
	})

	// Sign transaction
	signer := types.NewLondonSigner(l.chainID)
	signedTx, err := types.SignTx(tx, signer, key)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = l.client.SendTransaction(ctx, signedTx)
	if err != nil {
		// On error, we might need to refresh nonce
		// For simplicity, we just return the error
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	l.sentCount.Add(1)

	if l.callbacks != nil {
		if l.callbacks.OnSent != nil {
			l.callbacks.OnSent(signedTx.Hash())
		}
		if l.callbacks.OnTPS != nil {
			l.callbacks.OnTPS(l.getCurrentTPS())
		}
	}

	return nil
}

// getNonceAndIncrement atomically gets and increments the nonce for an account
func (l *LongSender) getNonceAndIncrement(accountIdx int) uint64 {
	return l.nonces[accountIdx].Add(1) - 1
}

// getCurrentTPS calculates the current TPS
func (l *LongSender) getCurrentTPS() float64 {
	elapsed := time.Since(l.startTime).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(l.sentCount.Load()) / elapsed
}

// recordError safely records an error
func (l *LongSender) recordError(err error) {
	l.errorsMu.Lock()
	defer l.errorsMu.Unlock()
	// Keep only last 100 errors
	if len(l.errors) < 100 {
		l.errors = append(l.errors, err)
	}
}

// GetStats returns current statistics
func (l *LongSender) GetStats() (sent, failed int64, tps float64) {
	return l.sentCount.Load(), l.failedCount.Load(), l.getCurrentTPS()
}
