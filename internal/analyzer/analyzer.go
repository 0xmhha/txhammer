package analyzer

import (
	"context"
	"encoding/csv"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"golang.org/x/sync/errgroup"
)

// Analyzer provides block analysis functionality
type Analyzer struct {
	client AnalyzerClient
	config *Config
	blocks []BlockInfo
	mu     sync.Mutex
}

// New creates a new Analyzer instance
func New(client AnalyzerClient, config *Config) *Analyzer {
	if config == nil {
		config = DefaultConfig()
	}
	if config.Concurrency <= 0 {
		config.Concurrency = runtime.NumCPU() * 10
	}
	return &Analyzer{
		client: client,
		config: config,
		blocks: make([]BlockInfo, 0),
	}
}

// Analyze performs block analysis and returns results
func (a *Analyzer) Analyze(ctx context.Context) (*AnalysisResult, error) {
	// Determine block range
	startBlock, endBlock, err := a.resolveBlockRange(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve block range: %w", err)
	}

	fmt.Printf("Analyzing blocks %d to %d (%d blocks)...\n", startBlock, endBlock, endBlock-startBlock+1)

	// Fetch blocks in parallel
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(a.config.Concurrency)

	for i := startBlock; i <= endBlock; i++ {
		blockNum := i // Capture for closure
		eg.Go(func() error {
			return a.fetchBlockInfo(egCtx, blockNum)
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("failed to fetch blocks: %w", err)
	}

	// Sort blocks by number
	a.sortBlocks()

	// Calculate metrics
	return a.calculateMetrics(), nil
}

// resolveBlockRange determines the actual block range to analyze
func (a *Analyzer) resolveBlockRange(ctx context.Context) (int64, int64, error) {
	var startBlock, endBlock int64

	// Get latest block if needed
	if a.config.EndBlock == 0 || a.config.BlockRange > 0 {
		latest, err := a.client.BlockNumber(ctx)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get latest block: %w", err)
		}
		endBlock = int64(latest)
	} else {
		endBlock = a.config.EndBlock
	}

	// Calculate start block
	if a.config.BlockRange > 0 {
		startBlock = endBlock - a.config.BlockRange + 1
		if startBlock < 1 {
			startBlock = 1
		}
	} else if a.config.StartBlock > 0 {
		startBlock = a.config.StartBlock
	} else {
		startBlock = 1
	}

	return startBlock, endBlock, nil
}

// fetchBlockInfo fetches information about a single block
func (a *Analyzer) fetchBlockInfo(ctx context.Context, blockNum int64) error {
	block, err := a.client.BlockByNumber(ctx, big.NewInt(blockNum))
	if err != nil {
		return fmt.Errorf("failed to fetch block %d: %w", blockNum, err)
	}

	utilization := float64(0)
	if block.GasLimit() > 0 {
		utilization = float64(block.GasUsed()) / float64(block.GasLimit()) * 100
	}

	info := BlockInfo{
		Number:      block.NumberU64(),
		Timestamp:   time.Unix(int64(block.Time()), 0),
		TxCount:     len(block.Transactions()),
		GasLimit:    block.GasLimit(),
		GasUsed:     block.GasUsed(),
		Utilization: utilization,
	}

	a.mu.Lock()
	a.blocks = append(a.blocks, info)
	a.mu.Unlock()

	return nil
}

// sortBlocks sorts blocks by number and calculates block times
func (a *Analyzer) sortBlocks() {
	sort.Slice(a.blocks, func(i, j int) bool {
		return a.blocks[i].Number < a.blocks[j].Number
	})

	// Calculate block times
	for i := 1; i < len(a.blocks); i++ {
		a.blocks[i].BlockTime = a.blocks[i].Timestamp.Sub(a.blocks[i-1].Timestamp)
	}
}

// calculateMetrics calculates aggregate metrics
func (a *Analyzer) calculateMetrics() *AnalysisResult {
	if len(a.blocks) == 0 {
		return &AnalysisResult{}
	}

	result := &AnalysisResult{
		StartBlock:    a.blocks[0].Number,
		EndBlock:      a.blocks[len(a.blocks)-1].Number,
		Blocks:        a.blocks,
		MinTxPerBlock: a.blocks[0].TxCount,
		MaxTxPerBlock: a.blocks[0].TxCount,
	}

	var totalGasUsed uint64
	var totalBlockTime time.Duration

	for i, block := range a.blocks {
		result.TotalTxs += uint64(block.TxCount)
		totalGasUsed += block.GasUsed

		if block.TxCount < result.MinTxPerBlock {
			result.MinTxPerBlock = block.TxCount
		}
		if block.TxCount > result.MaxTxPerBlock {
			result.MaxTxPerBlock = block.TxCount
		}

		if i > 0 {
			totalBlockTime += block.BlockTime
		}
	}

	// Calculate averages
	blockCount := len(a.blocks)
	if blockCount > 0 {
		result.AvgTxPerBlock = float64(result.TotalTxs) / float64(blockCount)
		result.AvgGasUsed = float64(totalGasUsed) / float64(blockCount)
	}

	if blockCount > 1 {
		result.AvgBlockTime = totalBlockTime / time.Duration(blockCount-1)
		result.TotalDuration = a.blocks[len(a.blocks)-1].Timestamp.Sub(a.blocks[0].Timestamp)

		if result.TotalDuration.Seconds() > 0 {
			result.AverageTPS = float64(result.TotalTxs) / result.TotalDuration.Seconds()
		}
	}

	return result
}

// PrintTable prints the analysis results as a table
func (a *Analyzer) PrintTable(result *AnalysisResult) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Block", "Time", "TxCount", "Gas Used", "Gas Limit", "Utilization", "Block Time"})
	table.SetBorder(true)

	for _, block := range result.Blocks {
		blockTime := "-"
		if block.BlockTime > 0 {
			blockTime = fmt.Sprintf("%.2fs", block.BlockTime.Seconds())
		}

		table.Append([]string{
			fmt.Sprintf("%d", block.Number),
			block.Timestamp.Format("15:04:05"),
			fmt.Sprintf("%d", block.TxCount),
			fmt.Sprintf("%d", block.GasUsed),
			fmt.Sprintf("%d", block.GasLimit),
			fmt.Sprintf("%.2f%%", block.Utilization),
			blockTime,
		})
	}

	// Add footer with summary
	table.SetFooter([]string{
		"TOTAL",
		fmt.Sprintf("%.2fs", result.TotalDuration.Seconds()),
		fmt.Sprintf("%d", result.TotalTxs),
		"-",
		"-",
		fmt.Sprintf("TPS: %.2f", result.AverageTPS),
		fmt.Sprintf("Avg: %.2fs", result.AvgBlockTime.Seconds()),
	})

	table.Render()

	// Print summary
	fmt.Println()
	fmt.Printf("Summary:\n")
	fmt.Printf("  Block Range: %d - %d (%d blocks)\n", result.StartBlock, result.EndBlock, len(result.Blocks))
	fmt.Printf("  Total Duration: %s\n", result.TotalDuration)
	fmt.Printf("  Total Transactions: %d\n", result.TotalTxs)
	fmt.Printf("  Average TPS: %.2f\n", result.AverageTPS)
	fmt.Printf("  Avg Block Time: %.2fs\n", result.AvgBlockTime.Seconds())
	fmt.Printf("  Avg Tx/Block: %.2f (min: %d, max: %d)\n", result.AvgTxPerBlock, result.MinTxPerBlock, result.MaxTxPerBlock)
	fmt.Printf("  Avg Gas Used: %.0f\n", result.AvgGasUsed)
}

// ExportCSV exports the results to a CSV file
func (a *Analyzer) ExportCSV(result *AnalysisResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Block", "Timestamp", "TxCount", "GasUsed", "GasLimit", "Utilization", "BlockTime"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write rows
	for _, block := range result.Blocks {
		row := []string{
			fmt.Sprintf("%d", block.Number),
			block.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%d", block.TxCount),
			fmt.Sprintf("%d", block.GasUsed),
			fmt.Sprintf("%d", block.GasLimit),
			fmt.Sprintf("%.4f", block.Utilization),
			fmt.Sprintf("%.3f", block.BlockTime.Seconds()),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}
