package collector

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ExportFormat represents the export format
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
)

// Exporter handles report export functionality
type Exporter struct {
	outputDir string
}

// NewExporter creates a new Exporter
func NewExporter(outputDir string) *Exporter {
	return &Exporter{
		outputDir: outputDir,
	}
}

// Export exports the report to the specified format
func (e *Exporter) Export(report *Report, format ExportFormat) (string, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(e.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")

	switch format {
	case FormatJSON:
		return e.exportJSON(report, timestamp)
	case FormatCSV:
		return e.exportCSV(report, timestamp)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// exportJSON exports the report as JSON
func (e *Exporter) exportJSON(report *Report, timestamp string) (string, error) {
	filename := filepath.Join(e.outputDir, fmt.Sprintf("report_%s.json", timestamp))

	// Create JSON-serializable report
	jsonReport := e.createJSONReport(report)

	data, err := json.MarshalIndent(jsonReport, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write report: %w", err)
	}

	return filename, nil
}

// JSONReport is a JSON-serializable version of Report
type JSONReport struct {
	TestName  string      `json:"test_name"`
	StartTime string      `json:"start_time"`
	EndTime   string      `json:"end_time"`
	Duration  string      `json:"duration"`
	Summary   JSONSummary `json:"summary"`
	Latency   JSONLatency `json:"latency"`
	Gas       JSONGas     `json:"gas"`
	Blocks    JSONBlocks  `json:"blocks"`
}

// JSONSummary is a JSON-serializable summary
type JSONSummary struct {
	TotalSent      int     `json:"total_sent"`
	TotalConfirmed int     `json:"total_confirmed"`
	TotalFailed    int     `json:"total_failed"`
	TotalTimeout   int     `json:"total_timeout"`
	TotalPending   int     `json:"total_pending"`
	SuccessRate    float64 `json:"success_rate"`
	TPS            float64 `json:"tps"`
	ConfirmedTPS   float64 `json:"confirmed_tps"`
}

// JSONLatency is a JSON-serializable latency metrics
type JSONLatency struct {
	Average string         `json:"average"`
	Min     string         `json:"min"`
	Max     string         `json:"max"`
	P50     string         `json:"p50"`
	P95     string         `json:"p95"`
	P99     string         `json:"p99"`
	Histogram map[string]int `json:"histogram"`
}

// JSONGas is a JSON-serializable gas metrics
type JSONGas struct {
	TotalUsed   uint64 `json:"total_used"`
	AverageUsed uint64 `json:"average_used"`
	TotalCost   string `json:"total_cost"`
	AverageCost string `json:"average_cost"`
}

// JSONBlocks is a JSON-serializable block metrics
type JSONBlocks struct {
	Observed         int     `json:"observed"`
	AvgBlockTime     string  `json:"avg_block_time"`
	AvgTxPerBlock    float64 `json:"avg_tx_per_block"`
	AvgUtilization   float64 `json:"avg_utilization"`
	FirstBlockWithTx uint64  `json:"first_block_with_tx,omitempty"`
	LastBlockWithTx  uint64  `json:"last_block_with_tx,omitempty"`
	BlockSpan        int     `json:"block_span,omitempty"`
	BlocksWithOurTx  int     `json:"blocks_with_our_tx,omitempty"`
	BlockBasedTPS    float64 `json:"block_based_tps,omitempty"`
}

// createJSONReport creates a JSON-serializable report
func (e *Exporter) createJSONReport(report *Report) *JSONReport {
	jr := &JSONReport{
		TestName:  report.TestName,
		StartTime: report.StartTime.Format(time.RFC3339),
		EndTime:   report.EndTime.Format(time.RFC3339),
		Duration:  report.Duration.String(),
		Summary: JSONSummary{
			TotalSent:      report.Metrics.TotalSent,
			TotalConfirmed: report.Metrics.TotalConfirmed,
			TotalFailed:    report.Metrics.TotalFailed,
			TotalTimeout:   report.Metrics.TotalTimeout,
			TotalPending:   report.Metrics.TotalPending,
			SuccessRate:    report.Metrics.SuccessRate,
			TPS:            report.Metrics.TPS,
			ConfirmedTPS:   report.Metrics.ConfirmedTPS,
		},
		Latency: JSONLatency{
			Average:   report.Metrics.AvgLatency.String(),
			Min:       report.Metrics.MinLatency.String(),
			Max:       report.Metrics.MaxLatency.String(),
			P50:       report.Metrics.P50Latency.String(),
			P95:       report.Metrics.P95Latency.String(),
			P99:       report.Metrics.P99Latency.String(),
			Histogram: report.LatencyHistogram,
		},
		Gas: JSONGas{
			TotalUsed:   report.Metrics.TotalGasUsed,
			AverageUsed: report.Metrics.AvgGasUsed,
		},
		Blocks: JSONBlocks{
			Observed:         report.Metrics.BlocksObserved,
			AvgBlockTime:     report.Metrics.AvgBlockTime.String(),
			AvgTxPerBlock:    report.Metrics.AvgTxPerBlock,
			AvgUtilization:   report.Metrics.AvgUtilization,
			FirstBlockWithTx: report.Metrics.FirstBlockWithTx,
			LastBlockWithTx:  report.Metrics.LastBlockWithTx,
			BlockSpan:        report.Metrics.BlockSpan,
			BlocksWithOurTx:  report.Metrics.BlocksWithOurTx,
			BlockBasedTPS:    report.Metrics.BlockBasedTPS,
		},
	}

	if report.Metrics.TotalGasCost != nil {
		jr.Gas.TotalCost = report.Metrics.TotalGasCost.String()
	}
	if report.Metrics.AvgGasCost != nil {
		jr.Gas.AverageCost = report.Metrics.AvgGasCost.String()
	}

	return jr
}

// exportCSV exports the report as CSV files
func (e *Exporter) exportCSV(report *Report, timestamp string) (string, error) {
	// Create summary CSV
	summaryFile := filepath.Join(e.outputDir, fmt.Sprintf("summary_%s.csv", timestamp))
	if err := e.exportSummaryCSV(report, summaryFile); err != nil {
		return "", err
	}

	// Create transactions CSV
	txFile := filepath.Join(e.outputDir, fmt.Sprintf("transactions_%s.csv", timestamp))
	if err := e.exportTransactionsCSV(report, txFile); err != nil {
		return "", err
	}

	// Create blocks CSV if available
	if len(report.Blocks) > 0 {
		blocksFile := filepath.Join(e.outputDir, fmt.Sprintf("blocks_%s.csv", timestamp))
		if err := e.exportBlocksCSV(report, blocksFile); err != nil {
			return "", err
		}
	}

	return summaryFile, nil
}

// exportSummaryCSV exports summary as CSV
func (e *Exporter) exportSummaryCSV(report *Report, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header and values
	records := [][]string{
		{"Metric", "Value"},
		{"Test Name", report.TestName},
		{"Start Time", report.StartTime.Format(time.RFC3339)},
		{"End Time", report.EndTime.Format(time.RFC3339)},
		{"Duration", report.Duration.String()},
		{"Total Sent", fmt.Sprintf("%d", report.Metrics.TotalSent)},
		{"Total Confirmed", fmt.Sprintf("%d", report.Metrics.TotalConfirmed)},
		{"Total Failed", fmt.Sprintf("%d", report.Metrics.TotalFailed)},
		{"Total Timeout", fmt.Sprintf("%d", report.Metrics.TotalTimeout)},
		{"Success Rate", fmt.Sprintf("%.2f%%", report.Metrics.SuccessRate)},
		{"TPS (Sent)", fmt.Sprintf("%.2f", report.Metrics.TPS)},
		{"TPS (Confirmed)", fmt.Sprintf("%.2f", report.Metrics.ConfirmedTPS)},
		{"Block-Based TPS", fmt.Sprintf("%.2f", report.Metrics.BlockBasedTPS)},
		{"First Block", fmt.Sprintf("%d", report.Metrics.FirstBlockWithTx)},
		{"Last Block", fmt.Sprintf("%d", report.Metrics.LastBlockWithTx)},
		{"Block Span", fmt.Sprintf("%d", report.Metrics.BlockSpan)},
		{"Blocks w/ Our Tx", fmt.Sprintf("%d", report.Metrics.BlocksWithOurTx)},
		{"Avg Latency", report.Metrics.AvgLatency.String()},
		{"Min Latency", report.Metrics.MinLatency.String()},
		{"Max Latency", report.Metrics.MaxLatency.String()},
		{"P50 Latency", report.Metrics.P50Latency.String()},
		{"P95 Latency", report.Metrics.P95Latency.String()},
		{"P99 Latency", report.Metrics.P99Latency.String()},
		{"Total Gas Used", fmt.Sprintf("%d", report.Metrics.TotalGasUsed)},
		{"Avg Gas Used", fmt.Sprintf("%d", report.Metrics.AvgGasUsed)},
	}

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	return nil
}

// exportTransactionsCSV exports transactions as CSV
func (e *Exporter) exportTransactionsCSV(report *Report, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Hash", "From", "Nonce", "GasLimit", "SentAt", "ConfirmedAt", "Status", "Latency", "GasUsed", "Error"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write transactions
	for _, tx := range report.Transactions {
		var gasUsed string
		if tx.Receipt != nil {
			gasUsed = fmt.Sprintf("%d", tx.Receipt.GasUsed)
		}

		var errStr string
		if tx.Error != nil {
			errStr = tx.Error.Error()
		}

		record := []string{
			tx.Hash.Hex(),
			tx.From.Hex(),
			fmt.Sprintf("%d", tx.Nonce),
			fmt.Sprintf("%d", tx.GasLimit),
			tx.SentAt.Format(time.RFC3339Nano),
			tx.ConfirmedAt.Format(time.RFC3339Nano),
			tx.Status.String(),
			tx.Latency.String(),
			gasUsed,
			errStr,
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	return nil
}

// exportBlocksCSV exports blocks as CSV
func (e *Exporter) exportBlocksCSV(report *Report, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Number", "Hash", "Timestamp", "GasLimit", "GasUsed", "TxCount", "OurTxCount", "Utilization"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write blocks
	for _, block := range report.Blocks {
		record := []string{
			fmt.Sprintf("%d", block.Number),
			block.Hash.Hex(),
			block.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%d", block.GasLimit),
			fmt.Sprintf("%d", block.GasUsed),
			fmt.Sprintf("%d", block.TxCount),
			fmt.Sprintf("%d", block.OurTxCount),
			fmt.Sprintf("%.2f%%", block.Utilization),
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	return nil
}

// ExportAll exports the report in all formats
func (e *Exporter) ExportAll(report *Report) ([]string, error) {
	files := make([]string, 0)

	jsonFile, err := e.Export(report, FormatJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to export JSON: %w", err)
	}
	files = append(files, jsonFile)

	csvFile, err := e.Export(report, FormatCSV)
	if err != nil {
		return nil, fmt.Errorf("failed to export CSV: %w", err)
	}
	files = append(files, csvFile)

	return files, nil
}
