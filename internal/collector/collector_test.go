package collector

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// mockCollectorClient implements Client interface for testing
type mockCollectorClient struct {
	receipts    map[common.Hash]*types.Receipt
	blocks      map[uint64]*types.Block
	blockNumber uint64
	receiptErr  error
	blockErr    error
	blockNumErr error
}

func newMockCollectorClient() *mockCollectorClient {
	return &mockCollectorClient{
		receipts:    make(map[common.Hash]*types.Receipt),
		blocks:      make(map[uint64]*types.Block),
		blockNumber: 1000,
	}
}

var errReceiptNotFound = errors.New("receipt not found")

func (m *mockCollectorClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	if m.receiptErr != nil {
		return nil, m.receiptErr
	}
	if receipt, ok := m.receipts[txHash]; ok {
		return receipt, nil
	}
	return nil, errReceiptNotFound
}

func (m *mockCollectorClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	if m.blockErr != nil {
		return nil, m.blockErr
	}
	if number == nil {
		return nil, nil
	}
	if block, ok := m.blocks[number.Uint64()]; ok {
		return block, nil
	}
	// Return empty block
	header := &types.Header{
		Number:   number,
		Time:     uint64(time.Now().Unix()),
		GasLimit: 30000000,
		GasUsed:  15000000,
	}
	return types.NewBlock(header, nil, nil, nil), nil
}

func (m *mockCollectorClient) BlockNumber(ctx context.Context) (uint64, error) {
	if m.blockNumErr != nil {
		return 0, m.blockNumErr
	}
	return m.blockNumber, nil
}

func (m *mockCollectorClient) BatchCall(batch []rpc.BatchElem) error {
	return nil
}

func (m *mockCollectorClient) addReceipt(hash common.Hash, status, gasUsed uint64) {
	m.receipts[hash] = &types.Receipt{
		Status:            status,
		GasUsed:           gasUsed,
		EffectiveGasPrice: big.NewInt(1000000000),
		TxHash:            hash,
	}
}

// Tests for TxConfirmStatus
func TestTxConfirmStatus_String(t *testing.T) {
	tests := []struct {
		status TxConfirmStatus
		want   string
	}{
		{TxConfirmPending, "PENDING"},
		{TxConfirmSuccess, "SUCCESS"},
		{TxConfirmFailed, "FAILED"},
		{TxConfirmTimeout, "TIMEOUT"},
		{TxConfirmNotFound, "NOT_FOUND"},
		{TxConfirmStatus(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("TxConfirmStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Tests for Config
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.PollInterval != 500*time.Millisecond {
		t.Errorf("PollInterval = %v, want 500ms", cfg.PollInterval)
	}
	if cfg.ConfirmTimeout != 60*time.Second {
		t.Errorf("ConfirmTimeout = %v, want 60s", cfg.ConfirmTimeout)
	}
	if cfg.MaxConcurrent != 20 {
		t.Errorf("MaxConcurrent = %d, want 20", cfg.MaxConcurrent)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("BatchSize = %d, want 100", cfg.BatchSize)
	}
	if !cfg.BlockTrackingEnabled {
		t.Error("BlockTrackingEnabled should be true")
	}
	if cfg.BlockPollInterval != 1*time.Second {
		t.Errorf("BlockPollInterval = %v, want 1s", cfg.BlockPollInterval)
	}
}

// Tests for Collector
func TestNew(t *testing.T) {
	client := newMockCollectorClient()

	// With nil config
	c1 := New(client, nil)
	if c1.config == nil {
		t.Error("New() with nil config should use default config")
	}

	// With custom config
	customCfg := &Config{
		PollInterval:   100 * time.Millisecond,
		ConfirmTimeout: 30 * time.Second,
	}
	c2 := New(client, customCfg)
	if c2.config.PollInterval != 100*time.Millisecond {
		t.Error("New() should use provided config")
	}
}

func TestCollector_TrackTransaction(t *testing.T) {
	client := newMockCollectorClient()
	collector := New(client, DefaultConfig())

	hash := common.HexToHash("0x1234567890")
	from := common.HexToAddress("0xabcdef")
	nonce := uint64(5)
	gasLimit := uint64(21000)
	sentAt := time.Now()

	collector.TrackTransaction(hash, from, nonce, gasLimit, sentAt)

	if collector.GetPendingCount() != 1 {
		t.Errorf("PendingCount = %d, want 1", collector.GetPendingCount())
	}
}

func TestCollector_TrackTransactions(t *testing.T) {
	client := newMockCollectorClient()
	collector := New(client, DefaultConfig())

	txInfos := []*TxInfo{
		{Hash: common.HexToHash("0x1"), From: common.HexToAddress("0xa"), Nonce: 0},
		{Hash: common.HexToHash("0x2"), From: common.HexToAddress("0xb"), Nonce: 1},
		{Hash: common.HexToHash("0x3"), From: common.HexToAddress("0xc"), Nonce: 2},
	}

	collector.TrackTransactions(txInfos)

	if collector.GetPendingCount() != 3 {
		t.Errorf("PendingCount = %d, want 3", collector.GetPendingCount())
	}
}

func TestCollector_Collect_EmptyTxs(t *testing.T) {
	client := newMockCollectorClient()
	collector := New(client, DefaultConfig())

	report, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if report.TestName != "empty" {
		t.Errorf("TestName = %s, want empty", report.TestName)
	}
}

func TestCollector_Collect_WithReceipts(t *testing.T) {
	client := newMockCollectorClient()

	cfg := &Config{
		PollInterval:         10 * time.Millisecond,
		ConfirmTimeout:       1 * time.Second,
		MaxConcurrent:        5,
		BatchSize:            10,
		BlockTrackingEnabled: false,
	}
	collector := New(client, cfg)

	// Track transactions
	hash1 := common.HexToHash("0x1111")
	hash2 := common.HexToHash("0x2222")

	collector.TrackTransaction(hash1, common.Address{}, 0, 21000, time.Now())
	collector.TrackTransaction(hash2, common.Address{}, 1, 21000, time.Now())

	// Add successful receipts
	client.addReceipt(hash1, types.ReceiptStatusSuccessful, 21000)
	client.addReceipt(hash2, types.ReceiptStatusSuccessful, 21000)

	report, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if report.Metrics.TotalSent != 2 {
		t.Errorf("TotalSent = %d, want 2", report.Metrics.TotalSent)
	}
	if report.Metrics.TotalConfirmed != 2 {
		t.Errorf("TotalConfirmed = %d, want 2", report.Metrics.TotalConfirmed)
	}
	if report.Metrics.SuccessRate != 100.0 {
		t.Errorf("SuccessRate = %f, want 100", report.Metrics.SuccessRate)
	}
}

func TestCollector_Collect_WithFailedReceipts(t *testing.T) {
	client := newMockCollectorClient()

	cfg := &Config{
		PollInterval:         10 * time.Millisecond,
		ConfirmTimeout:       1 * time.Second,
		MaxConcurrent:        5,
		BatchSize:            10,
		BlockTrackingEnabled: false,
	}
	collector := New(client, cfg)

	hash := common.HexToHash("0x3333")
	collector.TrackTransaction(hash, common.Address{}, 0, 21000, time.Now())

	// Add failed receipt
	client.addReceipt(hash, types.ReceiptStatusFailed, 21000)

	report, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if report.Metrics.TotalFailed != 1 {
		t.Errorf("TotalFailed = %d, want 1", report.Metrics.TotalFailed)
	}
}

func TestCollector_Collect_Timeout(t *testing.T) {
	client := newMockCollectorClient()

	cfg := &Config{
		PollInterval:         10 * time.Millisecond,
		ConfirmTimeout:       50 * time.Millisecond,
		MaxConcurrent:        5,
		BatchSize:            10,
		BlockTrackingEnabled: false,
	}
	collector := New(client, cfg)

	// Track transaction without adding receipt (will timeout)
	hash := common.HexToHash("0x4444")
	collector.TrackTransaction(hash, common.Address{}, 0, 21000, time.Now())

	report, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if report.Metrics.TotalTimeout != 1 {
		t.Errorf("TotalTimeout = %d, want 1", report.Metrics.TotalTimeout)
	}
}

func TestCollector_GetCounts(t *testing.T) {
	client := newMockCollectorClient()
	collector := New(client, DefaultConfig())

	// Track some transactions
	collector.TrackTransaction(common.HexToHash("0x1"), common.Address{}, 0, 21000, time.Now())
	collector.TrackTransaction(common.HexToHash("0x2"), common.Address{}, 1, 21000, time.Now())

	if collector.GetPendingCount() != 2 {
		t.Errorf("GetPendingCount() = %d, want 2", collector.GetPendingCount())
	}
	if collector.GetConfirmedCount() != 0 {
		t.Errorf("GetConfirmedCount() = %d, want 0", collector.GetConfirmedCount())
	}
	if collector.GetFailedCount() != 0 {
		t.Errorf("GetFailedCount() = %d, want 0", collector.GetFailedCount())
	}
}

func TestCollector_Reset(t *testing.T) {
	client := newMockCollectorClient()
	collector := New(client, DefaultConfig())

	collector.TrackTransaction(common.HexToHash("0x1"), common.Address{}, 0, 21000, time.Now())
	collector.TrackTransaction(common.HexToHash("0x2"), common.Address{}, 1, 21000, time.Now())

	collector.Reset()

	if collector.GetPendingCount() != 0 {
		t.Errorf("GetPendingCount() after Reset = %d, want 0", collector.GetPendingCount())
	}
	if collector.GetConfirmedCount() != 0 {
		t.Errorf("GetConfirmedCount() after Reset = %d, want 0", collector.GetConfirmedCount())
	}
	if collector.GetFailedCount() != 0 {
		t.Errorf("GetFailedCount() after Reset = %d, want 0", collector.GetFailedCount())
	}
}

// Tests for latency calculations
func TestCollector_calculateAvgLatency(t *testing.T) {
	client := newMockCollectorClient()
	collector := New(client, DefaultConfig())

	latencies := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
	}

	avg := collector.calculateAvgLatency(latencies)
	expected := 200 * time.Millisecond

	if avg != expected {
		t.Errorf("calculateAvgLatency() = %v, want %v", avg, expected)
	}
}

func TestCollector_calculateMinMaxLatency(t *testing.T) {
	client := newMockCollectorClient()
	collector := New(client, DefaultConfig())

	tests := []struct {
		name      string
		latencies []time.Duration
		wantMin   time.Duration
		wantMax   time.Duration
	}{
		{
			name:      "normal case",
			latencies: []time.Duration{100 * time.Millisecond, 500 * time.Millisecond, 200 * time.Millisecond},
			wantMin:   100 * time.Millisecond,
			wantMax:   500 * time.Millisecond,
		},
		{
			name:      "single element",
			latencies: []time.Duration{100 * time.Millisecond},
			wantMin:   100 * time.Millisecond,
			wantMax:   100 * time.Millisecond,
		},
		{
			name:      "empty",
			latencies: []time.Duration{},
			wantMin:   0,
			wantMax:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMin, gotMax := collector.calculateMinMaxLatency(tt.latencies)
			if gotMin != tt.wantMin {
				t.Errorf("min = %v, want %v", gotMin, tt.wantMin)
			}
			if gotMax != tt.wantMax {
				t.Errorf("max = %v, want %v", gotMax, tt.wantMax)
			}
		})
	}
}

func TestCollector_calculatePercentile(t *testing.T) {
	client := newMockCollectorClient()
	collector := New(client, DefaultConfig())

	tests := []struct {
		name       string
		latencies  []time.Duration
		percentile int
	}{
		{
			name:       "p50",
			latencies:  []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond, 400 * time.Millisecond, 500 * time.Millisecond},
			percentile: 50,
		},
		{
			name:       "p95",
			latencies:  []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond, 400 * time.Millisecond, 500 * time.Millisecond},
			percentile: 95,
		},
		{
			name:       "empty",
			latencies:  []time.Duration{},
			percentile: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.calculatePercentile(tt.latencies, tt.percentile)
			// Just verify it doesn't panic
			_ = result
		})
	}
}

func TestCollector_buildLatencyHistogram(t *testing.T) {
	client := newMockCollectorClient()
	collector := New(client, DefaultConfig())

	latencies := []time.Duration{
		50 * time.Millisecond,   // <100ms
		150 * time.Millisecond,  // 100-500ms
		600 * time.Millisecond,  // 500ms-1s
		1500 * time.Millisecond, // 1-2s
		3 * time.Second,         // 2-5s
		10 * time.Second,        // >5s
	}

	histogram := collector.buildLatencyHistogram(latencies)

	if histogram["<100ms"] != 1 {
		t.Errorf("<100ms bucket = %d, want 1", histogram["<100ms"])
	}
	if histogram["100-500ms"] != 1 {
		t.Errorf("100-500ms bucket = %d, want 1", histogram["100-500ms"])
	}
	if histogram["500ms-1s"] != 1 {
		t.Errorf("500ms-1s bucket = %d, want 1", histogram["500ms-1s"])
	}
	if histogram["1-2s"] != 1 {
		t.Errorf("1-2s bucket = %d, want 1", histogram["1-2s"])
	}
	if histogram["2-5s"] != 1 {
		t.Errorf("2-5s bucket = %d, want 1", histogram["2-5s"])
	}
}

// Tests for Report
func TestNewReport(t *testing.T) {
	report := NewReport("test-report")

	if report.TestName != "test-report" {
		t.Errorf("TestName = %s, want test-report", report.TestName)
	}
	if report.Metrics == nil {
		t.Error("Metrics should not be nil")
	}
	if report.Transactions == nil {
		t.Error("Transactions should not be nil")
	}
	if report.Blocks == nil {
		t.Error("Blocks should not be nil")
	}
	if report.LatencyHistogram == nil {
		t.Error("LatencyHistogram should not be nil")
	}
	if report.ErrorSummary == nil {
		t.Error("ErrorSummary should not be nil")
	}
}

// Tests for TxInfo
func TestTxInfo(t *testing.T) {
	hash := common.HexToHash("0x1234")
	from := common.HexToAddress("0xabcd")
	sentAt := time.Now()
	confirmedAt := sentAt.Add(1 * time.Second)

	info := &TxInfo{
		Hash:        hash,
		From:        from,
		Nonce:       10,
		GasLimit:    21000,
		SentAt:      sentAt,
		ConfirmedAt: confirmedAt,
		Status:      TxConfirmSuccess,
		Latency:     1 * time.Second,
	}

	if info.Hash != hash {
		t.Errorf("Hash mismatch")
	}
	if info.From != from {
		t.Errorf("From mismatch")
	}
	if info.Nonce != 10 {
		t.Errorf("Nonce = %d, want 10", info.Nonce)
	}
	if info.GasLimit != 21000 {
		t.Errorf("GasLimit = %d, want 21000", info.GasLimit)
	}
	if info.SentAt.IsZero() {
		t.Error("SentAt should be set")
	}
	if info.ConfirmedAt.IsZero() {
		t.Error("ConfirmedAt should be set")
	}
	if info.Status != TxConfirmSuccess {
		t.Errorf("Status = %v, want SUCCESS", info.Status)
	}
	if info.Latency <= 0 {
		t.Errorf("Latency should be positive, got %s", info.Latency)
	}
}

// Tests for BlockInfo
func TestBlockInfo(t *testing.T) {
	hash := common.HexToHash("0x5678")

	block := &BlockInfo{
		Number:      100,
		Hash:        hash,
		Timestamp:   time.Now(),
		GasLimit:    30000000,
		GasUsed:     15000000,
		TxCount:     100,
		OurTxCount:  50,
		BaseFee:     big.NewInt(1000000000),
		Utilization: 50.0,
	}

	if block.Number != 100 {
		t.Errorf("Number = %d, want 100", block.Number)
	}
	if block.Hash != hash {
		t.Errorf("Hash mismatch")
	}
	if block.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
	if block.GasLimit != 30000000 {
		t.Errorf("GasLimit = %d, want 30000000", block.GasLimit)
	}
	if block.GasUsed != 15000000 {
		t.Errorf("GasUsed = %d, want 15000000", block.GasUsed)
	}
	if block.TxCount != 100 {
		t.Errorf("TxCount = %d, want 100", block.TxCount)
	}
	if block.OurTxCount != 50 {
		t.Errorf("OurTxCount = %d, want 50", block.OurTxCount)
	}
	if block.BaseFee == nil || block.BaseFee.Sign() <= 0 {
		t.Error("BaseFee should be set")
	}
	if block.Utilization != 50.0 {
		t.Errorf("Utilization = %f, want 50.0", block.Utilization)
	}
}

// Tests for Metrics
func TestMetrics(t *testing.T) {
	metrics := &Metrics{
		TotalSent:      100,
		TotalConfirmed: 95,
		TotalFailed:    3,
		TotalPending:   1,
		TotalTimeout:   1,
		TPS:            50.0,
		ConfirmedTPS:   47.5,
		SuccessRate:    95.0,
		TotalGasUsed:   2100000,
		AvgGasUsed:     21000,
		BlocksObserved: 10,
	}

	if metrics.TotalSent != 100 {
		t.Errorf("TotalSent = %d, want 100", metrics.TotalSent)
	}
	if metrics.TotalConfirmed != 95 {
		t.Errorf("TotalConfirmed = %d, want 95", metrics.TotalConfirmed)
	}
	if metrics.TotalFailed != 3 {
		t.Errorf("TotalFailed = %d, want 3", metrics.TotalFailed)
	}
	if metrics.TotalPending != 1 {
		t.Errorf("TotalPending = %d, want 1", metrics.TotalPending)
	}
	if metrics.TotalTimeout != 1 {
		t.Errorf("TotalTimeout = %d, want 1", metrics.TotalTimeout)
	}
	if metrics.TPS != 50.0 {
		t.Errorf("TPS = %f, want 50.0", metrics.TPS)
	}
	if metrics.ConfirmedTPS != 47.5 {
		t.Errorf("ConfirmedTPS = %f, want 47.5", metrics.ConfirmedTPS)
	}
	if metrics.SuccessRate != 95.0 {
		t.Errorf("SuccessRate = %f, want 95.0", metrics.SuccessRate)
	}
	if metrics.TotalGasUsed != 2100000 {
		t.Errorf("TotalGasUsed = %d, want 2100000", metrics.TotalGasUsed)
	}
	if metrics.AvgGasUsed != 21000 {
		t.Errorf("AvgGasUsed = %d, want 21000", metrics.AvgGasUsed)
	}
	if metrics.BlocksObserved != 10 {
		t.Errorf("BlocksObserved = %d, want 10", metrics.BlocksObserved)
	}
}

// Tests for Config fields
func TestConfig(t *testing.T) {
	cfg := &Config{
		PollInterval:         100 * time.Millisecond,
		ConfirmTimeout:       30 * time.Second,
		MaxConcurrent:        10,
		BatchSize:            50,
		BlockTrackingEnabled: true,
		BlockPollInterval:    2 * time.Second,
	}

	if cfg.PollInterval != 100*time.Millisecond {
		t.Errorf("PollInterval = %v, want 100ms", cfg.PollInterval)
	}
	if cfg.ConfirmTimeout != 30*time.Second {
		t.Errorf("ConfirmTimeout = %v, want 30s", cfg.ConfirmTimeout)
	}
	if cfg.MaxConcurrent != 10 {
		t.Errorf("MaxConcurrent = %d, want 10", cfg.MaxConcurrent)
	}
	if cfg.BatchSize != 50 {
		t.Errorf("BatchSize = %d, want 50", cfg.BatchSize)
	}
	if !cfg.BlockTrackingEnabled {
		t.Error("BlockTrackingEnabled should be true")
	}
	if cfg.BlockPollInterval != 2*time.Second {
		t.Errorf("BlockPollInterval = %v, want 2s", cfg.BlockPollInterval)
	}
}
