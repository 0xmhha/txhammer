package batcher

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/txhammer/internal/txbuilder"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	testPrivateKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

// mockBatchClient implements Client interface for testing
type mockBatchClient struct {
	mu              sync.Mutex
	batchSendResult []common.Hash
	batchSendErr    error
	batchCallErr    error
	callCount       int
}

func (m *mockBatchClient) BatchSendRawTransactions(ctx context.Context, rawTxs [][]byte) ([]common.Hash, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	if m.batchSendErr != nil {
		return nil, m.batchSendErr
	}
	if m.batchSendResult != nil {
		return m.batchSendResult, nil
	}
	// Generate hashes for each raw tx
	hashes := make([]common.Hash, len(rawTxs))
	for i := range rawTxs {
		hashes[i] = crypto.Keccak256Hash(rawTxs[i])
	}
	return hashes, nil
}

func (m *mockBatchClient) BatchCall(batch []rpc.BatchElem) error {
	return m.batchCallErr
}

// mockStreamClient implements StreamClient interface for testing
type mockStreamClient struct {
	mu         sync.Mutex
	sendResult common.Hash
	sendErr    error
	callCount  int
}

func (m *mockStreamClient) SendRawTransaction(ctx context.Context, rawTx []byte) (common.Hash, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	if m.sendErr != nil {
		return common.Hash{}, m.sendErr
	}
	if m.sendResult != (common.Hash{}) {
		return m.sendResult, nil
	}
	return crypto.Keccak256Hash(rawTx), nil
}

func createTestTxs(count int) []*txbuilder.SignedTx {
	key, _ := crypto.HexToECDSA(testPrivateKey)
	addr := crypto.PubkeyToAddress(key.PublicKey)

	txs := make([]*txbuilder.SignedTx, count)
	for i := 0; i < count; i++ {
		txs[i] = &txbuilder.SignedTx{
			RawTx:    []byte{byte(i), byte(i + 1), byte(i + 2)},
			Hash:     crypto.Keccak256Hash([]byte{byte(i)}),
			From:     addr,
			Nonce:    uint64(i),
			GasLimit: 21000,
		}
	}
	return txs
}

// Tests for TxStatus
func TestTxStatus_String(t *testing.T) {
	tests := []struct {
		status TxStatus
		want   string
	}{
		{TxStatusPending, "PENDING"},
		{TxStatusSent, "SENT"},
		{TxStatusConfirmed, "CONFIRMED"},
		{TxStatusFailed, "FAILED"},
		{TxStatus(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("TxStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Tests for Config
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BatchSize != 100 {
		t.Errorf("BatchSize = %d, want 100", cfg.BatchSize)
	}
	if cfg.MaxConcurrent != 5 {
		t.Errorf("MaxConcurrent = %d, want 5", cfg.MaxConcurrent)
	}
	if cfg.BatchInterval != 100*time.Millisecond {
		t.Errorf("BatchInterval = %v, want 100ms", cfg.BatchInterval)
	}
	if cfg.RetryCount != 3 {
		t.Errorf("RetryCount = %d, want 3", cfg.RetryCount)
	}
	if cfg.RetryDelay != 500*time.Millisecond {
		t.Errorf("RetryDelay = %v, want 500ms", cfg.RetryDelay)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		check  func(*Config) bool
	}{
		{
			name:   "zero batch size gets default",
			config: &Config{BatchSize: 0},
			check:  func(c *Config) bool { return c.BatchSize == 100 },
		},
		{
			name:   "negative batch size gets default",
			config: &Config{BatchSize: -1},
			check:  func(c *Config) bool { return c.BatchSize == 100 },
		},
		{
			name:   "zero max concurrent gets default",
			config: &Config{MaxConcurrent: 0},
			check:  func(c *Config) bool { return c.MaxConcurrent == 5 },
		},
		{
			name:   "negative batch interval gets default",
			config: &Config{BatchInterval: -1},
			check:  func(c *Config) bool { return c.BatchInterval == 100*time.Millisecond },
		},
		{
			name:   "negative retry count gets zero",
			config: &Config{RetryCount: -1},
			check:  func(c *Config) bool { return c.RetryCount == 0 },
		},
		{
			name:   "zero timeout gets default",
			config: &Config{Timeout: 0},
			check:  func(c *Config) bool { return c.Timeout == 30*time.Second },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != nil {
				t.Errorf("Validate() error = %v", err)
			}
			if !tt.check(tt.config) {
				t.Errorf("Validate() did not set correct default")
			}
		})
	}
}

// Tests for Batcher
func TestNew(t *testing.T) {
	client := &mockBatchClient{}

	// With nil config
	b1 := New(client, nil)
	if b1.config == nil {
		t.Error("New() with nil config should use default config")
	}

	// With custom config
	customCfg := &Config{
		BatchSize:     50,
		MaxConcurrent: 10,
	}
	b2 := New(client, customCfg)
	if b2.config.BatchSize != 50 {
		t.Error("New() should use provided config")
	}
}

func TestBatcher_SendAll_EmptyTxs(t *testing.T) {
	client := &mockBatchClient{}
	batcher := New(client, DefaultConfig())

	summary, err := batcher.SendAll(context.Background(), nil)
	if err != nil {
		t.Fatalf("SendAll() error = %v", err)
	}

	if summary.TotalTxs != 0 {
		t.Errorf("TotalTxs = %d, want 0", summary.TotalTxs)
	}
}

func TestBatcher_SendAll_Success(t *testing.T) {
	client := &mockBatchClient{}
	cfg := &Config{
		BatchSize:     10,
		MaxConcurrent: 2,
		BatchInterval: 0,
		RetryCount:    0,
		Timeout:       5 * time.Second,
	}
	batcher := New(client, cfg)

	txs := createTestTxs(25)

	summary, err := batcher.SendAll(context.Background(), txs)
	if err != nil {
		t.Fatalf("SendAll() error = %v", err)
	}

	if summary.TotalTxs != 25 {
		t.Errorf("TotalTxs = %d, want 25", summary.TotalTxs)
	}
	if summary.SuccessCount != 25 {
		t.Errorf("SuccessCount = %d, want 25", summary.SuccessCount)
	}
	if summary.FailedCount != 0 {
		t.Errorf("FailedCount = %d, want 0", summary.FailedCount)
	}
	if summary.TotalBatches != 3 {
		t.Errorf("TotalBatches = %d, want 3", summary.TotalBatches)
	}
}

func TestBatcher_SendAll_WithFailures(t *testing.T) {
	client := &mockBatchClient{
		batchSendErr: errors.New("batch send failed"),
	}
	cfg := &Config{
		BatchSize:     10,
		MaxConcurrent: 1,
		BatchInterval: 0,
		RetryCount:    0,
		Timeout:       1 * time.Second,
	}
	batcher := New(client, cfg)

	txs := createTestTxs(10)

	summary, err := batcher.SendAll(context.Background(), txs)
	if err != nil {
		t.Fatalf("SendAll() error = %v", err)
	}

	if summary.FailedCount != 10 {
		t.Errorf("FailedCount = %d, want 10", summary.FailedCount)
	}
	if summary.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0", summary.SuccessCount)
	}
	if len(summary.FailedTxs) != 10 {
		t.Errorf("FailedTxs = %d, want 10", len(summary.FailedTxs))
	}
}

func TestBatcher_splitIntoBatches(t *testing.T) {
	client := &mockBatchClient{}
	cfg := &Config{BatchSize: 10}
	batcher := New(client, cfg)

	tests := []struct {
		name        string
		txCount     int
		wantBatches int
	}{
		{"exact division", 30, 3},
		{"with remainder", 25, 3},
		{"single batch", 5, 1},
		{"empty", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txs := createTestTxs(tt.txCount)
			batches := batcher.splitIntoBatches(txs)

			if len(batches) != tt.wantBatches {
				t.Errorf("splitIntoBatches() = %d batches, want %d", len(batches), tt.wantBatches)
			}

			// Verify total tx count
			total := 0
			for _, batch := range batches {
				total += len(batch)
			}
			if total != tt.txCount {
				t.Errorf("total txs = %d, want %d", total, tt.txCount)
			}
		})
	}
}

func TestBatcher_GetSentCount(t *testing.T) {
	client := &mockBatchClient{}
	cfg := &Config{
		BatchSize:     10,
		MaxConcurrent: 1,
		BatchInterval: 0,
		RetryCount:    0,
		Timeout:       5 * time.Second,
	}
	batcher := New(client, cfg)

	txs := createTestTxs(15)
	_, err := batcher.SendAll(context.Background(), txs)
	if err != nil {
		t.Fatalf("SendAll() error = %v", err)
	}

	if batcher.GetSentCount() != 15 {
		t.Errorf("GetSentCount() = %d, want 15", batcher.GetSentCount())
	}
}

func TestBatcher_GetFailedCount(t *testing.T) {
	client := &mockBatchClient{
		batchSendErr: errors.New("failed"),
	}
	cfg := &Config{
		BatchSize:     10,
		MaxConcurrent: 1,
		BatchInterval: 0,
		RetryCount:    0,
		Timeout:       1 * time.Second,
	}
	batcher := New(client, cfg)

	txs := createTestTxs(10)
	_, _ = batcher.SendAll(context.Background(), txs)

	if batcher.GetFailedCount() != 10 {
		t.Errorf("GetFailedCount() = %d, want 10", batcher.GetFailedCount())
	}
}

func TestBatcher_Reset(t *testing.T) {
	client := &mockBatchClient{}
	cfg := &Config{
		BatchSize:     10,
		MaxConcurrent: 1,
		BatchInterval: 0,
		RetryCount:    0,
		Timeout:       5 * time.Second,
	}
	batcher := New(client, cfg)

	txs := createTestTxs(10)
	_, _ = batcher.SendAll(context.Background(), txs)

	batcher.Reset()

	if batcher.GetSentCount() != 0 {
		t.Errorf("GetSentCount() after Reset = %d, want 0", batcher.GetSentCount())
	}
	if batcher.GetFailedCount() != 0 {
		t.Errorf("GetFailedCount() after Reset = %d, want 0", batcher.GetFailedCount())
	}
}

// retryMockClient implements Client with retry behavior
type retryMockClient struct {
	callCount      int
	failUntilCount int
}

func (m *retryMockClient) BatchSendRawTransactions(ctx context.Context, rawTxs [][]byte) ([]common.Hash, error) {
	m.callCount++
	if m.callCount < m.failUntilCount {
		return nil, errors.New("temporary failure")
	}
	// Generate hashes for each raw tx
	hashes := make([]common.Hash, len(rawTxs))
	for i := range rawTxs {
		hashes[i] = crypto.Keccak256Hash(rawTxs[i])
	}
	return hashes, nil
}

func (m *retryMockClient) BatchCall(batch []rpc.BatchElem) error {
	return nil
}

func TestBatcher_sendBatchWithRetry(t *testing.T) {
	// Create a client that fails first 2 times, succeeds on 3rd
	client := &retryMockClient{
		failUntilCount: 3,
	}

	cfg := &Config{
		BatchSize:     10,
		MaxConcurrent: 1,
		BatchInterval: 0,
		RetryCount:    3,
		RetryDelay:    10 * time.Millisecond,
		Timeout:       5 * time.Second,
	}
	batcher := New(client, cfg)

	rawTxs := [][]byte{{0x01}, {0x02}}
	hashes, err := batcher.sendBatchWithRetry(context.Background(), rawTxs)

	if err != nil {
		t.Fatalf("sendBatchWithRetry() error = %v", err)
	}
	if len(hashes) != 2 {
		t.Errorf("hashes count = %d, want 2", len(hashes))
	}
	if client.callCount != 3 {
		t.Errorf("callCount = %d, want 3", client.callCount)
	}
}

// Tests for Streamer
func TestDefaultStreamerConfig(t *testing.T) {
	cfg := DefaultStreamerConfig()

	if cfg.Rate != 1000 {
		t.Errorf("Rate = %f, want 1000", cfg.Rate)
	}
	if cfg.Burst != 100 {
		t.Errorf("Burst = %d, want 100", cfg.Burst)
	}
	if cfg.Workers != 10 {
		t.Errorf("Workers = %d, want 10", cfg.Workers)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", cfg.Timeout)
	}
}

func TestNewStreamer(t *testing.T) {
	client := &mockStreamClient{}

	// With nil config
	s1 := NewStreamer(client, nil)
	if s1.config == nil {
		t.Error("NewStreamer() with nil config should use default config")
	}

	// With custom config
	customCfg := &StreamerConfig{
		Rate:    500,
		Burst:   50,
		Workers: 5,
	}
	s2 := NewStreamer(client, customCfg)
	if s2.config.Rate != 500 {
		t.Error("NewStreamer() should use provided config")
	}
}

func TestStreamer_Stream_EmptyTxs(t *testing.T) {
	client := &mockStreamClient{}
	streamer := NewStreamer(client, DefaultStreamerConfig())

	result, err := streamer.Stream(context.Background(), nil)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	if result.TotalTxs != 0 {
		t.Errorf("TotalTxs = %d, want 0", result.TotalTxs)
	}
}

func TestStreamer_Stream_Success(t *testing.T) {
	client := &mockStreamClient{}
	cfg := &StreamerConfig{
		Rate:    10000, // High rate for fast test
		Burst:   100,
		Workers: 5,
		Timeout: 5 * time.Second,
	}
	streamer := NewStreamer(client, cfg)

	txs := createTestTxs(10)

	result, err := streamer.Stream(context.Background(), txs)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	if result.TotalTxs != 10 {
		t.Errorf("TotalTxs = %d, want 10", result.TotalTxs)
	}
	if result.SuccessCount != 10 {
		t.Errorf("SuccessCount = %d, want 10", result.SuccessCount)
	}
	if result.FailedCount != 0 {
		t.Errorf("FailedCount = %d, want 0", result.FailedCount)
	}
}

func TestStreamer_Stream_WithFailures(t *testing.T) {
	client := &mockStreamClient{
		sendErr: errors.New("send failed"),
	}
	cfg := &StreamerConfig{
		Rate:    10000,
		Burst:   100,
		Workers: 5,
		Timeout: 1 * time.Second,
	}
	streamer := NewStreamer(client, cfg)

	txs := createTestTxs(5)

	result, err := streamer.Stream(context.Background(), txs)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	if result.FailedCount != 5 {
		t.Errorf("FailedCount = %d, want 5", result.FailedCount)
	}
	if result.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0", result.SuccessCount)
	}
	if len(result.FailedTxs) != 5 {
		t.Errorf("FailedTxs = %d, want 5", len(result.FailedTxs))
	}
}

func TestStreamer_GetSentCount(t *testing.T) {
	client := &mockStreamClient{}
	cfg := &StreamerConfig{
		Rate:    10000,
		Burst:   100,
		Workers: 5,
		Timeout: 5 * time.Second,
	}
	streamer := NewStreamer(client, cfg)

	txs := createTestTxs(10)
	_, _ = streamer.Stream(context.Background(), txs)

	if streamer.GetSentCount() != 10 {
		t.Errorf("GetSentCount() = %d, want 10", streamer.GetSentCount())
	}
}

func TestStreamer_GetFailedCount(t *testing.T) {
	client := &mockStreamClient{
		sendErr: errors.New("failed"),
	}
	cfg := &StreamerConfig{
		Rate:    10000,
		Burst:   100,
		Workers: 5,
		Timeout: 1 * time.Second,
	}
	streamer := NewStreamer(client, cfg)

	txs := createTestTxs(5)
	_, _ = streamer.Stream(context.Background(), txs)

	if streamer.GetFailedCount() != 5 {
		t.Errorf("GetFailedCount() = %d, want 5", streamer.GetFailedCount())
	}
}

func TestStreamer_Reset(t *testing.T) {
	client := &mockStreamClient{}
	cfg := &StreamerConfig{
		Rate:    10000,
		Burst:   100,
		Workers: 5,
		Timeout: 5 * time.Second,
	}
	streamer := NewStreamer(client, cfg)

	txs := createTestTxs(5)
	_, _ = streamer.Stream(context.Background(), txs)

	streamer.Reset()

	if streamer.GetSentCount() != 0 {
		t.Errorf("GetSentCount() after Reset = %d, want 0", streamer.GetSentCount())
	}
	if streamer.GetFailedCount() != 0 {
		t.Errorf("GetFailedCount() after Reset = %d, want 0", streamer.GetFailedCount())
	}
}

// Test types
func TestBatchResult(t *testing.T) {
	result := &BatchResult{
		BatchIndex:   0,
		TxCount:      10,
		SuccessCount: 8,
		FailedCount:  2,
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(1 * time.Second),
		Duration:     1 * time.Second,
	}

	if result.BatchIndex != 0 {
		t.Errorf("BatchIndex = %d, want 0", result.BatchIndex)
	}
	if result.TxCount != 10 {
		t.Errorf("TxCount = %d, want 10", result.TxCount)
	}
}

func TestSummary(t *testing.T) {
	summary := &Summary{
		TotalBatches:  5,
		TotalTxs:      100,
		SuccessCount:  95,
		FailedCount:   5,
		TotalDuration: 10 * time.Second,
		AvgBatchTime:  2 * time.Second,
		TxPerSecond:   9.5,
	}

	if summary.TotalBatches != 5 {
		t.Errorf("TotalBatches = %d, want 5", summary.TotalBatches)
	}
	if summary.TxPerSecond != 9.5 {
		t.Errorf("TxPerSecond = %f, want 9.5", summary.TxPerSecond)
	}
}

func TestStreamResult(t *testing.T) {
	result := &StreamResult{
		TotalTxs:      50,
		SuccessCount:  48,
		FailedCount:   2,
		TotalDuration: 5 * time.Second,
		TxPerSecond:   9.6,
	}

	if result.TotalTxs != 50 {
		t.Errorf("TotalTxs = %d, want 50", result.TotalTxs)
	}
	if result.FailedCount != 2 {
		t.Errorf("FailedCount = %d, want 2", result.FailedCount)
	}
}

// Test TxResult
func TestTxResult(t *testing.T) {
	tx := createTestTxs(1)[0]
	result := &TxResult{
		Tx:       tx,
		Hash:     common.HexToHash("0x1234"),
		Status:   TxStatusSent,
		SentAt:   time.Now(),
		BatchIdx: 0,
	}

	if result.Status != TxStatusSent {
		t.Errorf("Status = %v, want SENT", result.Status)
	}
	if result.Hash != common.HexToHash("0x1234") {
		t.Errorf("Hash mismatch")
	}
}

// Prevent unused import error
var _ = big.NewInt(1)
