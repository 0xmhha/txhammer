package collector

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/schollz/progressbar/v3"
)

// Client interface for collector operations
type Client interface {
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	BlockNumber(ctx context.Context) (uint64, error)
	BatchCall(batch []rpc.BatchElem) error
}

// Collector handles transaction receipt collection and metrics
type Collector struct {
	client Client
	config *Config

	// Tracking state
	txMap    map[common.Hash]*TxInfo
	txMutex  sync.RWMutex
	blocks   []*BlockInfo
	blockMu  sync.RWMutex

	// Metrics
	confirmed atomic.Int64
	failed    atomic.Int64
	pending   atomic.Int64
}

// New creates a new Collector instance
func New(client Client, config *Config) *Collector {
	if config == nil {
		config = DefaultConfig()
	}

	return &Collector{
		client: client,
		config: config,
		txMap:  make(map[common.Hash]*TxInfo),
		blocks: make([]*BlockInfo, 0),
	}
}

// TrackTransaction adds a transaction to be tracked
func (c *Collector) TrackTransaction(hash common.Hash, from common.Address, nonce, gasLimit uint64, sentAt time.Time) {
	c.txMutex.Lock()
	defer c.txMutex.Unlock()

	c.txMap[hash] = &TxInfo{
		Hash:     hash,
		From:     from,
		Nonce:    nonce,
		GasLimit: gasLimit,
		SentAt:   sentAt,
		Status:   TxConfirmPending,
	}
	c.pending.Add(1)
}

// TrackTransactions adds multiple transactions to be tracked
func (c *Collector) TrackTransactions(txInfos []*TxInfo) {
	c.txMutex.Lock()
	defer c.txMutex.Unlock()

	for _, info := range txInfos {
		info.Status = TxConfirmPending
		c.txMap[info.Hash] = info
		c.pending.Add(1)
	}
}

// Collect starts the collection process and waits for all transactions
func (c *Collector) Collect(ctx context.Context) (*Report, error) {
	c.txMutex.RLock()
	totalTxs := len(c.txMap)
	c.txMutex.RUnlock()

	if totalTxs == 0 {
		return NewReport("empty"), nil
	}

	fmt.Printf("\nStarting Receipt Collection\n\n")
	fmt.Printf("Total transactions to collect: %d\n", totalTxs)
	fmt.Printf("Poll interval: %s\n", c.config.PollInterval)
	fmt.Printf("Confirm timeout: %s\n\n", c.config.ConfirmTimeout)

	report := NewReport("stress-test")

	// Create progress bar
	bar := progressbar.Default(int64(totalTxs), "collecting receipts")

	// Start block tracking if enabled
	var blockCtx context.Context
	var blockCancel context.CancelFunc
	if c.config.BlockTrackingEnabled {
		blockCtx, blockCancel = context.WithCancel(ctx)
		go c.trackBlocks(blockCtx)
	}

	// Collection loop
	deadline := time.Now().Add(c.config.ConfirmTimeout)
	collected := 0

	for collected < totalTxs {
		if time.Now().After(deadline) {
			// Mark remaining as timeout
			c.markTimeouts()
			break
		}

		select {
		case <-ctx.Done():
			if blockCancel != nil {
				blockCancel()
			}
			return nil, ctx.Err()
		default:
		}

		// Collect pending receipts
		newCollected := c.collectBatch(ctx)
		if newCollected > 0 {
			_ = bar.Add(newCollected)
			collected += newCollected
		}

		time.Sleep(c.config.PollInterval)
	}

	if blockCancel != nil {
		blockCancel()
	}

	fmt.Println()

	// Build report
	report = c.buildReport(report)

	// Print summary
	c.printSummary(report)

	return report, nil
}

// collectBatch collects receipts for pending transactions
func (c *Collector) collectBatch(ctx context.Context) int {
	// Get pending transactions
	c.txMutex.RLock()
	pending := make([]*TxInfo, 0)
	for _, tx := range c.txMap {
		if tx.Status == TxConfirmPending {
			pending = append(pending, tx)
			if len(pending) >= c.config.BatchSize {
				break
			}
		}
	}
	c.txMutex.RUnlock()

	if len(pending) == 0 {
		return 0
	}

	// Query receipts concurrently
	var wg sync.WaitGroup
	sem := make(chan struct{}, c.config.MaxConcurrent)
	collected := atomic.Int32{}

	for _, txInfo := range pending {
		wg.Add(1)
		go func(info *TxInfo) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			receipt, err := c.client.TransactionReceipt(ctx, info.Hash)
			if err != nil {
				// Not yet mined, keep pending
				return
			}

			c.txMutex.Lock()
			info.ConfirmedAt = time.Now()
			info.Latency = info.ConfirmedAt.Sub(info.SentAt)
			info.Receipt = receipt

			if receipt.Status == types.ReceiptStatusSuccessful {
				info.Status = TxConfirmSuccess
				c.confirmed.Add(1)
			} else {
				info.Status = TxConfirmFailed
				c.failed.Add(1)
			}
			c.pending.Add(-1)
			c.txMutex.Unlock()

			collected.Add(1)
		}(txInfo)
	}

	wg.Wait()
	return int(collected.Load())
}

// markTimeouts marks remaining pending transactions as timeout
func (c *Collector) markTimeouts() {
	c.txMutex.Lock()
	defer c.txMutex.Unlock()

	for _, tx := range c.txMap {
		if tx.Status == TxConfirmPending {
			tx.Status = TxConfirmTimeout
			tx.Error = fmt.Errorf("confirmation timeout")
			c.pending.Add(-1)
		}
	}
}

// trackBlocks tracks block-level metrics
func (c *Collector) trackBlocks(ctx context.Context) {
	ticker := time.NewTicker(c.config.BlockPollInterval)
	defer ticker.Stop()

	var lastBlock uint64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			blockNum, err := c.client.BlockNumber(ctx)
			if err != nil {
				continue
			}

			if blockNum > lastBlock {
				// Fetch new blocks
				for num := lastBlock + 1; num <= blockNum; num++ {
					block, err := c.client.BlockByNumber(ctx, big.NewInt(int64(num)))
					if err != nil {
						continue
					}

					blockInfo := &BlockInfo{
						Number:    num,
						Hash:      block.Hash(),
						Timestamp: time.Unix(int64(block.Time()), 0),
						GasLimit:  block.GasLimit(),
						GasUsed:   block.GasUsed(),
						TxCount:   len(block.Transactions()),
						BaseFee:   block.BaseFee(),
					}

					if blockInfo.GasLimit > 0 {
						blockInfo.Utilization = float64(blockInfo.GasUsed) / float64(blockInfo.GasLimit) * 100
					}

					// Count our transactions in this block
					c.txMutex.RLock()
					for _, tx := range block.Transactions() {
						if _, exists := c.txMap[tx.Hash()]; exists {
							blockInfo.OurTxCount++
						}
					}
					c.txMutex.RUnlock()

					c.blockMu.Lock()
					c.blocks = append(c.blocks, blockInfo)
					c.blockMu.Unlock()
				}
				lastBlock = blockNum
			}
		}
	}
}

// buildReport builds the final report from collected data
func (c *Collector) buildReport(report *Report) *Report {
	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)

	c.txMutex.RLock()
	defer c.txMutex.RUnlock()

	c.blockMu.RLock()
	defer c.blockMu.RUnlock()

	// Copy transactions
	latencies := make([]time.Duration, 0)
	var totalGasUsed uint64
	totalGasCost := big.NewInt(0)

	for _, tx := range c.txMap {
		report.Transactions = append(report.Transactions, tx)

		switch tx.Status {
		case TxConfirmSuccess:
			report.Metrics.TotalConfirmed++
			latencies = append(latencies, tx.Latency)
			if tx.Receipt != nil {
				totalGasUsed += tx.Receipt.GasUsed
				cost := new(big.Int).Mul(
					big.NewInt(int64(tx.Receipt.GasUsed)),
					tx.Receipt.EffectiveGasPrice,
				)
				totalGasCost.Add(totalGasCost, cost)
			}
		case TxConfirmFailed:
			report.Metrics.TotalFailed++
			if tx.Error != nil {
				errStr := tx.Error.Error()
				report.ErrorSummary[errStr]++
			}
		case TxConfirmPending:
			report.Metrics.TotalPending++
		case TxConfirmTimeout:
			report.Metrics.TotalTimeout++
		}
	}

	report.Metrics.TotalSent = len(c.txMap)
	report.Metrics.EndTime = report.EndTime
	report.Metrics.TotalDuration = report.Duration

	// Calculate latency metrics
	if len(latencies) > 0 {
		report.Metrics.AvgLatency = c.calculateAvgLatency(latencies)
		report.Metrics.MinLatency, report.Metrics.MaxLatency = c.calculateMinMaxLatency(latencies)
		report.Metrics.P50Latency = c.calculatePercentile(latencies, 50)
		report.Metrics.P95Latency = c.calculatePercentile(latencies, 95)
		report.Metrics.P99Latency = c.calculatePercentile(latencies, 99)
		report.LatencyHistogram = c.buildLatencyHistogram(latencies)
	}

	// Calculate TPS
	if report.Duration.Seconds() > 0 {
		report.Metrics.TPS = float64(report.Metrics.TotalSent) / report.Duration.Seconds()
		report.Metrics.ConfirmedTPS = float64(report.Metrics.TotalConfirmed) / report.Duration.Seconds()
	}

	// Calculate gas metrics
	if report.Metrics.TotalConfirmed > 0 {
		report.Metrics.TotalGasUsed = totalGasUsed
		report.Metrics.AvgGasUsed = totalGasUsed / uint64(report.Metrics.TotalConfirmed)
		report.Metrics.TotalGasCost = totalGasCost
		report.Metrics.AvgGasCost = new(big.Int).Div(totalGasCost, big.NewInt(int64(report.Metrics.TotalConfirmed)))
	}

	// Calculate success rate
	if report.Metrics.TotalSent > 0 {
		report.Metrics.SuccessRate = float64(report.Metrics.TotalConfirmed) / float64(report.Metrics.TotalSent) * 100
	}

	// Copy blocks and calculate block metrics
	report.Blocks = c.blocks
	report.Metrics.BlocksObserved = len(c.blocks)

	if len(c.blocks) > 1 {
		var totalBlockTime time.Duration
		var totalTxPerBlock float64
		var totalUtilization float64

		for i := 1; i < len(c.blocks); i++ {
			blockTime := c.blocks[i].Timestamp.Sub(c.blocks[i-1].Timestamp)
			totalBlockTime += blockTime
		}
		report.Metrics.AvgBlockTime = totalBlockTime / time.Duration(len(c.blocks)-1)

		for _, block := range c.blocks {
			totalTxPerBlock += float64(block.TxCount)
			totalUtilization += block.Utilization
		}
		report.Metrics.AvgTxPerBlock = totalTxPerBlock / float64(len(c.blocks))
		report.Metrics.AvgUtilization = totalUtilization / float64(len(c.blocks))
	}

	// Calculate block-based TPS (transactions per block span)
	// Find first and last blocks containing our transactions, and count blocks with our txs
	var firstBlock, lastBlock uint64
	var foundFirst bool
	blocksWithOurTx := 0
	for _, block := range c.blocks {
		if block.OurTxCount > 0 {
			blocksWithOurTx++
			if !foundFirst {
				firstBlock = block.Number
				foundFirst = true
			}
			lastBlock = block.Number
		}
	}

	if foundFirst && lastBlock >= firstBlock {
		report.Metrics.FirstBlockWithTx = firstBlock
		report.Metrics.LastBlockWithTx = lastBlock
		report.Metrics.BlockSpan = int(lastBlock-firstBlock) + 1
		report.Metrics.BlocksWithOurTx = blocksWithOurTx

		// Calculate TPS based on blocks that contain our transactions
		// TPS = TotalConfirmed / (BlocksWithOurTx × AvgBlockTime)
		if blocksWithOurTx > 0 && report.Metrics.AvgBlockTime.Seconds() > 0 {
			report.Metrics.BlockBasedTPS = float64(report.Metrics.TotalConfirmed) / (float64(blocksWithOurTx) * report.Metrics.AvgBlockTime.Seconds())
		}
	}

	return report
}

// calculateAvgLatency calculates average latency
func (c *Collector) calculateAvgLatency(latencies []time.Duration) time.Duration {
	var total time.Duration
	for _, l := range latencies {
		total += l
	}
	return total / time.Duration(len(latencies))
}

// calculateMinMaxLatency calculates min and max latency
func (c *Collector) calculateMinMaxLatency(latencies []time.Duration) (time.Duration, time.Duration) {
	if len(latencies) == 0 {
		return 0, 0
	}

	min, max := latencies[0], latencies[0]
	for _, l := range latencies[1:] {
		if l < min {
			min = l
		}
		if l > max {
			max = l
		}
	}
	return min, max
}

// calculatePercentile calculates latency percentile
func (c *Collector) calculatePercentile(latencies []time.Duration, p int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Simple percentile calculation (not sorted for efficiency)
	// In production, should use proper sorting or reservoir sampling
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Simple bubble sort for small datasets
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	idx := (len(sorted) - 1) * p / 100
	return sorted[idx]
}

// buildLatencyHistogram builds latency distribution histogram
func (c *Collector) buildLatencyHistogram(latencies []time.Duration) map[string]int {
	histogram := make(map[string]int)
	buckets := []struct {
		label string
		max   time.Duration
	}{
		{"<100ms", 100 * time.Millisecond},
		{"100-500ms", 500 * time.Millisecond},
		{"500ms-1s", 1 * time.Second},
		{"1-2s", 2 * time.Second},
		{"2-5s", 5 * time.Second},
		{">5s", 0},
	}

	for _, l := range latencies {
		for i, bucket := range buckets {
			if bucket.max == 0 || l < bucket.max {
				histogram[bucket.label]++
				break
			}
			if i == len(buckets)-1 {
				histogram[bucket.label]++
			}
		}
	}

	return histogram
}

// printSummary prints the collection summary
func (c *Collector) printSummary(report *Report) {
	fmt.Printf("\nCollection Summary\n\n")

	// Transaction summary
	fmt.Printf("Transactions:\n")
	fmt.Printf("  Total Sent:      %d\n", report.Metrics.TotalSent)
	fmt.Printf("  Confirmed:       %d (%.2f%%)\n", report.Metrics.TotalConfirmed, report.Metrics.SuccessRate)
	fmt.Printf("  Failed:          %d\n", report.Metrics.TotalFailed)
	fmt.Printf("  Timeout:         %d\n", report.Metrics.TotalTimeout)
	fmt.Printf("  Pending:         %d\n", report.Metrics.TotalPending)

	// Timing
	fmt.Printf("\nTiming:\n")
	fmt.Printf("  Total Duration:  %s\n", report.Duration)
	fmt.Printf("  TPS (sent):      %.2f\n", report.Metrics.TPS)
	fmt.Printf("  TPS (confirmed): %.2f\n", report.Metrics.ConfirmedTPS)

	// Latency
	if report.Metrics.TotalConfirmed > 0 {
		fmt.Printf("\nLatency:\n")
		fmt.Printf("  Average:         %s\n", report.Metrics.AvgLatency)
		fmt.Printf("  Min:             %s\n", report.Metrics.MinLatency)
		fmt.Printf("  Max:             %s\n", report.Metrics.MaxLatency)
		fmt.Printf("  P50:             %s\n", report.Metrics.P50Latency)
		fmt.Printf("  P95:             %s\n", report.Metrics.P95Latency)
		fmt.Printf("  P99:             %s\n", report.Metrics.P99Latency)
	}

	// Gas
	if report.Metrics.TotalGasUsed > 0 {
		fmt.Printf("\nGas:\n")
		fmt.Printf("  Total Used:      %d\n", report.Metrics.TotalGasUsed)
		fmt.Printf("  Average Used:    %d\n", report.Metrics.AvgGasUsed)
		fmt.Printf("  Total Cost:      %s wei\n", report.Metrics.TotalGasCost.String())
	}

	// Blocks
	if report.Metrics.BlocksObserved > 0 {
		fmt.Printf("\nBlocks:\n")
		fmt.Printf("  Observed:        %d\n", report.Metrics.BlocksObserved)
		fmt.Printf("  Avg Block Time:  %s\n", report.Metrics.AvgBlockTime)
		fmt.Printf("  Avg Tx/Block:    %.2f\n", report.Metrics.AvgTxPerBlock)
		fmt.Printf("  Avg Utilization: %.2f%%\n", report.Metrics.AvgUtilization)

		// Block-based TPS (real throughput)
		if report.Metrics.BlockSpan > 0 {
			fmt.Printf("\nBlock-Based Throughput:\n")
			fmt.Printf("  First Block:     #%d\n", report.Metrics.FirstBlockWithTx)
			fmt.Printf("  Last Block:      #%d\n", report.Metrics.LastBlockWithTx)
			fmt.Printf("  Block Span:      %d blocks\n", report.Metrics.BlockSpan)
			fmt.Printf("  Blocks w/ Tx:    %d blocks\n", report.Metrics.BlocksWithOurTx)
			fmt.Printf("  Block-Based TPS: %.2f tx/s\n", report.Metrics.BlockBasedTPS)
		}
	}

	// Latency histogram
	if len(report.LatencyHistogram) > 0 {
		fmt.Printf("\nLatency Distribution:\n")
		bucketOrder := []string{"<100ms", "100-500ms", "500ms-1s", "1-2s", "2-5s", ">5s"}
		for _, bucket := range bucketOrder {
			if count, ok := report.LatencyHistogram[bucket]; ok {
				pct := float64(count) / float64(report.Metrics.TotalConfirmed) * 100
				fmt.Printf("  %-12s %5d (%.1f%%)\n", bucket, count, pct)
			}
		}
	}

	// Errors
	if len(report.ErrorSummary) > 0 {
		fmt.Printf("\n[WARN] Errors:\n")
		for errMsg, count := range report.ErrorSummary {
			if len(errMsg) > 50 {
				errMsg = errMsg[:47] + "..."
			}
			fmt.Printf("  %s: %d\n", errMsg, count)
		}
	}
}

// GetConfirmedCount returns the number of confirmed transactions
func (c *Collector) GetConfirmedCount() int64 {
	return c.confirmed.Load()
}

// GetFailedCount returns the number of failed transactions
func (c *Collector) GetFailedCount() int64 {
	return c.failed.Load()
}

// GetPendingCount returns the number of pending transactions
func (c *Collector) GetPendingCount() int64 {
	return c.pending.Load()
}

// Reset resets the collector state
func (c *Collector) Reset() {
	c.txMutex.Lock()
	c.txMap = make(map[common.Hash]*TxInfo)
	c.txMutex.Unlock()

	c.blockMu.Lock()
	c.blocks = make([]*BlockInfo, 0)
	c.blockMu.Unlock()

	c.confirmed.Store(0)
	c.failed.Store(0)
	c.pending.Store(0)
}
