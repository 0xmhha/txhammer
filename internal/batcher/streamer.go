package batcher

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/0xmhha/txhammer/internal/txbuilder"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/time/rate"
)

// StreamerConfig holds streamer configuration
type StreamerConfig struct {
	// Rate is the number of transactions per second
	Rate float64

	// Burst is the maximum burst size
	Burst int

	// Workers is the number of concurrent workers
	Workers int

	// Timeout per transaction
	Timeout time.Duration
}

// DefaultStreamerConfig returns default streamer configuration
func DefaultStreamerConfig() *StreamerConfig {
	return &StreamerConfig{
		Rate:    1000,           // 1000 tx/s
		Burst:   100,            // burst of 100
		Workers: 10,             // 10 workers
		Timeout: 5 * time.Second,
	}
}

// StreamClient interface for streaming operations
type StreamClient interface {
	SendRawTransaction(ctx context.Context, rawTx []byte) (common.Hash, error)
}

// Streamer sends transactions in a streaming fashion with rate limiting
type Streamer struct {
	client  StreamClient
	config  *StreamerConfig
	limiter *rate.Limiter

	// Metrics
	sentCount   atomic.Int64
	failedCount atomic.Int64
}

// NewStreamer creates a new Streamer instance
func NewStreamer(client StreamClient, config *StreamerConfig) *Streamer {
	if config == nil {
		config = DefaultStreamerConfig()
	}

	return &Streamer{
		client:  client,
		config:  config,
		limiter: rate.NewLimiter(rate.Limit(config.Rate), config.Burst),
	}
}

// StreamResult represents the result of streaming operation
type StreamResult struct {
	TotalTxs      int
	SuccessCount  int
	FailedCount   int
	TotalDuration time.Duration
	TxPerSecond   float64
	Results       []*TxResult
	FailedTxs     []*TxResult
}

// Stream sends all transactions with rate limiting
func (s *Streamer) Stream(ctx context.Context, txs []*txbuilder.SignedTx) (*StreamResult, error) {
	if len(txs) == 0 {
		return &StreamResult{}, nil
	}

	fmt.Printf("\nStarting Streaming Transaction Sending\n\n")
	fmt.Printf("Total transactions: %d\n", len(txs))
	fmt.Printf("Rate limit: %.0f tx/s\n", s.config.Rate)
	fmt.Printf("Workers: %d\n", s.config.Workers)
	fmt.Printf("Burst: %d\n\n", s.config.Burst)

	startTime := time.Now()

	// Create progress bar
	bar := progressbar.Default(int64(len(txs)), "streaming txs")

	// Create result channels
	results := make([]*TxResult, len(txs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.config.Workers)

	for i, tx := range txs {
		// Wait for rate limiter
		if err := s.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter error: %w", err)
		}

		wg.Add(1)
		go func(idx int, signedTx *txbuilder.SignedTx) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			result := s.sendSingle(ctx, signedTx)
			results[idx] = result

			_ = bar.Add(1)
		}(i, tx)
	}

	wg.Wait()
	fmt.Println()

	// Build result
	totalDuration := time.Since(startTime)
	streamResult := s.buildResult(results, totalDuration)

	// Print summary
	s.printSummary(streamResult)

	return streamResult, nil
}

// sendSingle sends a single transaction
func (s *Streamer) sendSingle(ctx context.Context, tx *txbuilder.SignedTx) *TxResult {
	result := &TxResult{
		Tx:     tx,
		Status: TxStatusPending,
	}

	// Create timeout context
	sendCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	hash, err := s.client.SendRawTransaction(sendCtx, tx.RawTx)
	result.SentAt = time.Now()

	if err != nil {
		result.Status = TxStatusFailed
		result.Error = err
		s.failedCount.Add(1)
	} else {
		result.Hash = hash
		result.Status = TxStatusSent
		s.sentCount.Add(1)
	}

	return result
}

// buildResult builds the stream result
func (s *Streamer) buildResult(results []*TxResult, duration time.Duration) *StreamResult {
	sr := &StreamResult{
		TotalTxs:      len(results),
		TotalDuration: duration,
		Results:       results,
		FailedTxs:     make([]*TxResult, 0),
	}

	for _, r := range results {
		if r.Status == TxStatusFailed {
			sr.FailedCount++
			sr.FailedTxs = append(sr.FailedTxs, r)
		} else {
			sr.SuccessCount++
		}
	}

	if duration.Seconds() > 0 {
		sr.TxPerSecond = float64(sr.SuccessCount) / duration.Seconds()
	}

	return sr
}

// printSummary prints the streaming summary
func (s *Streamer) printSummary(result *StreamResult) {
	fmt.Printf("\nStreaming Summary\n\n")
	fmt.Printf("Total transactions: %d\n", result.TotalTxs)
	fmt.Printf("Successful: %d (%.2f%%)\n", result.SuccessCount,
		float64(result.SuccessCount)/float64(result.TotalTxs)*100)
	fmt.Printf("Failed: %d (%.2f%%)\n", result.FailedCount,
		float64(result.FailedCount)/float64(result.TotalTxs)*100)
	fmt.Printf("Total duration: %s\n", result.TotalDuration)
	fmt.Printf("Actual throughput: %.2f tx/s\n", result.TxPerSecond)

	if len(result.FailedTxs) > 0 {
		fmt.Printf("\n[WARN] Failed Transactions: %d\n", len(result.FailedTxs))
		showCount := 5
		if len(result.FailedTxs) < showCount {
			showCount = len(result.FailedTxs)
		}
		for i := 0; i < showCount; i++ {
			ft := result.FailedTxs[i]
			fmt.Printf("  - From: %s, Error: %v\n", ft.Tx.From.Hex()[:10], ft.Error)
		}
		if len(result.FailedTxs) > showCount {
			fmt.Printf("  ... and %d more\n", len(result.FailedTxs)-showCount)
		}
	}
}

// GetSentCount returns the number of successfully sent transactions
func (s *Streamer) GetSentCount() int64 {
	return s.sentCount.Load()
}

// GetFailedCount returns the number of failed transactions
func (s *Streamer) GetFailedCount() int64 {
	return s.failedCount.Load()
}

// Reset resets the streamer metrics
func (s *Streamer) Reset() {
	s.sentCount.Store(0)
	s.failedCount.Store(0)
}
