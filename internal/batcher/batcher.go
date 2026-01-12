package batcher

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/piatoss3612/txhammer/internal/txbuilder"
	"github.com/schollz/progressbar/v3"
)

// Client interface for batch operations
type Client interface {
	BatchSendRawTransactions(ctx context.Context, rawTxs [][]byte) ([]common.Hash, error)
	BatchCall(batch []rpc.BatchElem) error
}

// Batcher handles batch transaction sending
type Batcher struct {
	client Client
	config *Config

	// Metrics
	sentCount   atomic.Int64
	failedCount atomic.Int64
}

// New creates a new Batcher instance
func New(client Client, config *Config) *Batcher {
	if config == nil {
		config = DefaultConfig()
	}
	_ = config.Validate()

	return &Batcher{
		client: client,
		config: config,
	}
}

// SendAll sends all transactions in batches
func (b *Batcher) SendAll(ctx context.Context, txs []*txbuilder.SignedTx) (*Summary, error) {
	if len(txs) == 0 {
		return &Summary{}, nil
	}

	fmt.Printf("\n🚀 Starting Batch Transaction Sending 🚀\n\n")
	fmt.Printf("Total transactions: %d\n", len(txs))
	fmt.Printf("Batch size: %d\n", b.config.BatchSize)
	fmt.Printf("Max concurrent: %d\n", b.config.MaxConcurrent)
	fmt.Printf("Batch interval: %s\n\n", b.config.BatchInterval)

	startTime := time.Now()

	// Split into batches
	batches := b.splitIntoBatches(txs)
	fmt.Printf("Total batches: %d\n\n", len(batches))

	// Create progress bar
	bar := progressbar.Default(int64(len(txs)), "sending txs")

	// Process batches with concurrency control
	batchResults := make([]*BatchResult, len(batches))
	var wg sync.WaitGroup
	sem := make(chan struct{}, b.config.MaxConcurrent)

	for i, batch := range batches {
		wg.Add(1)
		go func(idx int, batchTxs []*txbuilder.SignedTx) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			result := b.sendBatch(ctx, idx, batchTxs)
			batchResults[idx] = result

			// Update progress
			_ = bar.Add(len(batchTxs))

			// Wait between batches
			if b.config.BatchInterval > 0 {
				time.Sleep(b.config.BatchInterval)
			}
		}(i, batch)
	}

	wg.Wait()
	fmt.Println()

	// Build summary
	summary := b.buildSummary(batchResults, time.Since(startTime))

	// Print summary
	b.printSummary(summary)

	return summary, nil
}

// splitIntoBatches splits transactions into batches
func (b *Batcher) splitIntoBatches(txs []*txbuilder.SignedTx) [][]*txbuilder.SignedTx {
	var batches [][]*txbuilder.SignedTx

	for i := 0; i < len(txs); i += b.config.BatchSize {
		end := i + b.config.BatchSize
		if end > len(txs) {
			end = len(txs)
		}
		batches = append(batches, txs[i:end])
	}

	return batches
}

// sendBatch sends a single batch of transactions
func (b *Batcher) sendBatch(ctx context.Context, batchIdx int, txs []*txbuilder.SignedTx) *BatchResult {
	startTime := time.Now()

	result := &BatchResult{
		BatchIndex: batchIdx,
		TxCount:    len(txs),
		StartTime:  startTime,
		Results:    make([]*TxResult, len(txs)),
	}

	// Prepare raw transactions
	rawTxs := make([][]byte, len(txs))
	for i, tx := range txs {
		rawTxs[i] = tx.RawTx
		result.Results[i] = &TxResult{
			Tx:       tx,
			Status:   TxStatusPending,
			BatchIdx: batchIdx,
		}
	}

	// Create timeout context
	sendCtx, cancel := context.WithTimeout(ctx, b.config.Timeout)
	defer cancel()

	// Send batch
	hashes, err := b.sendBatchWithRetry(sendCtx, rawTxs)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)

	if err != nil {
		result.Error = err
		// Mark all as failed
		for i := range result.Results {
			result.Results[i].Status = TxStatusFailed
			result.Results[i].Error = err
			result.FailedCount++
			b.failedCount.Add(1)
		}
		return result
	}

	// Process results
	now := time.Now()
	for i, hash := range hashes {
		result.Results[i].Hash = hash
		result.Results[i].SentAt = now

		if hash == (common.Hash{}) {
			result.Results[i].Status = TxStatusFailed
			result.FailedCount++
			b.failedCount.Add(1)
		} else {
			result.Results[i].Status = TxStatusSent
			result.SuccessCount++
			b.sentCount.Add(1)
		}
	}

	return result
}

// sendBatchWithRetry sends a batch with retry logic
func (b *Batcher) sendBatchWithRetry(ctx context.Context, rawTxs [][]byte) ([]common.Hash, error) {
	var lastErr error

	for attempt := 0; attempt <= b.config.RetryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(b.config.RetryDelay)
		}

		hashes, err := b.client.BatchSendRawTransactions(ctx, rawTxs)
		if err == nil {
			return hashes, nil
		}

		lastErr = err

		// Check if context is cancelled
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("batch send failed after %d retries: %w", b.config.RetryCount+1, lastErr)
}

// buildSummary builds the summary from batch results
func (b *Batcher) buildSummary(batchResults []*BatchResult, totalDuration time.Duration) *Summary {
	summary := &Summary{
		TotalBatches:  len(batchResults),
		TotalDuration: totalDuration,
		BatchResults:  batchResults,
		FailedTxs:     make([]*TxResult, 0),
	}

	var totalBatchTime time.Duration

	for _, br := range batchResults {
		summary.TotalTxs += br.TxCount
		summary.SuccessCount += br.SuccessCount
		summary.FailedCount += br.FailedCount
		totalBatchTime += br.Duration

		// Collect failed transactions
		for _, tr := range br.Results {
			if tr.Status == TxStatusFailed {
				summary.FailedTxs = append(summary.FailedTxs, tr)
			}
		}
	}

	if len(batchResults) > 0 {
		summary.AvgBatchTime = totalBatchTime / time.Duration(len(batchResults))
	}

	if totalDuration.Seconds() > 0 {
		summary.TxPerSecond = float64(summary.SuccessCount) / totalDuration.Seconds()
	}

	return summary
}

// printSummary prints the batch operation summary
func (b *Batcher) printSummary(summary *Summary) {
	fmt.Printf("\n📊 Batch Sending Summary 📊\n\n")
	fmt.Printf("Total batches: %d\n", summary.TotalBatches)
	fmt.Printf("Total transactions: %d\n", summary.TotalTxs)
	fmt.Printf("Successful: %d (%.2f%%)\n", summary.SuccessCount,
		float64(summary.SuccessCount)/float64(summary.TotalTxs)*100)
	fmt.Printf("Failed: %d (%.2f%%)\n", summary.FailedCount,
		float64(summary.FailedCount)/float64(summary.TotalTxs)*100)
	fmt.Printf("Total duration: %s\n", summary.TotalDuration)
	fmt.Printf("Avg batch time: %s\n", summary.AvgBatchTime)
	fmt.Printf("Throughput: %.2f tx/s\n", summary.TxPerSecond)

	if len(summary.FailedTxs) > 0 {
		fmt.Printf("\n⚠️  Failed Transactions: %d\n", len(summary.FailedTxs))
		// Show first 5 failed txs
		showCount := 5
		if len(summary.FailedTxs) < showCount {
			showCount = len(summary.FailedTxs)
		}
		for i := 0; i < showCount; i++ {
			ft := summary.FailedTxs[i]
			fmt.Printf("  - Batch %d, From: %s, Error: %v\n",
				ft.BatchIdx, ft.Tx.From.Hex()[:10], ft.Error)
		}
		if len(summary.FailedTxs) > showCount {
			fmt.Printf("  ... and %d more\n", len(summary.FailedTxs)-showCount)
		}
	}
}

// GetSentCount returns the number of successfully sent transactions
func (b *Batcher) GetSentCount() int64 {
	return b.sentCount.Load()
}

// GetFailedCount returns the number of failed transactions
func (b *Batcher) GetFailedCount() int64 {
	return b.failedCount.Load()
}

// Reset resets the batcher metrics
func (b *Batcher) Reset() {
	b.sentCount.Store(0)
	b.failedCount.Store(0)
}
