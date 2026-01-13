package pipeline

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/0xmhha/txhammer/internal/analyzer"
	"github.com/0xmhha/txhammer/internal/batcher"
	"github.com/0xmhha/txhammer/internal/client"
	"github.com/0xmhha/txhammer/internal/collector"
	"github.com/0xmhha/txhammer/internal/config"
	"github.com/0xmhha/txhammer/internal/distributor"
	"github.com/0xmhha/txhammer/internal/longsender"
	"github.com/0xmhha/txhammer/internal/metrics"
	"github.com/0xmhha/txhammer/internal/monitor"
	"github.com/0xmhha/txhammer/internal/txbuilder"
	"github.com/0xmhha/txhammer/internal/wallet"
)

// Pipeline orchestrates the stress test execution
type Pipeline struct {
	cfg       *config.Config
	runCfg    *RunConfig
	client    *client.Client
	wallet    *wallet.Wallet
	chainID   *big.Int

	// Components
	distributor *distributor.Distributor
	builder     txbuilder.Builder
	batcher     *batcher.Batcher
	streamer    *batcher.Streamer
	collector   *collector.Collector

	// State
	signedTxs []*txbuilder.SignedTx
	nonces    []uint64
}

// New creates a new pipeline instance
func New(cfg *config.Config) (*Pipeline, error) {
	// Create RPC client
	cli, err := client.New(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Create wallet
	var w *wallet.Wallet
	if cfg.Mnemonic != "" {
		w, err = wallet.NewFromMnemonic(cfg.Mnemonic, cfg.SubAccounts)
	} else {
		w, err = wallet.NewFromPrivateKey(cfg.PrivateKey, cfg.SubAccounts)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	return &Pipeline{
		cfg:    cfg,
		runCfg: DefaultRunConfig(),
		client: cli,
		wallet: w,
	}, nil
}

// WithRunConfig sets the run configuration
func (p *Pipeline) WithRunConfig(runCfg *RunConfig) *Pipeline {
	p.runCfg = runCfg
	return p
}

// Execute runs the complete stress test pipeline
func (p *Pipeline) Execute(ctx context.Context) (*Result, error) {
	result := NewResult()

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                          TxHammer                             ║")
	fmt.Println("║              StableNet Stress Testing Tool                     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Start Prometheus metrics server if enabled
	var metricsServer *metrics.Metrics
	if p.cfg.MetricsEnabled {
		metricsServer = metrics.NewMetrics("txhammer")
		if err := metricsServer.Start(ctx, p.cfg.MetricsPort); err != nil {
			fmt.Printf("[WARN] Failed to start metrics server: %v\n", err)
		} else {
			fmt.Printf("Prometheus metrics available at http://localhost:%d/metrics\n", p.cfg.MetricsPort)
		}
		defer func() {
			_ = metricsServer.Stop(ctx)
		}()
	}

	// Handle special modes
	mode := p.cfg.GetMode()
	switch mode {
	case config.ModeAnalyzeBlocks:
		return p.executeAnalyzeBlocks(ctx, result)
	case config.ModeLongSender:
		return p.executeLongSender(ctx, result, metricsServer)
	}

	// Stage 1: Initialize
	if err := p.runStage(ctx, result, StageInit, p.initialize); err != nil {
		return result, err
	}

	// Stage 2: Distribute funds
	if !p.runCfg.SkipDistribution {
		if err := p.runStage(ctx, result, StageDistribute, p.distribute); err != nil {
			return result, err
		}
	}

	// Stage 3: Build transactions
	if err := p.runStage(ctx, result, StageBuild, p.build); err != nil {
		return result, err
	}

	// Dry run - stop here
	if p.runCfg.DryRun {
		fmt.Println("\nDry run complete - transactions built but not sent")
		result.Finalize()
		return result, nil
	}

	// Stage 4: Send transactions
	if err := p.runStage(ctx, result, StageSend, p.send); err != nil {
		return result, err
	}

	// Stage 5: Collect results
	if !p.runCfg.SkipCollection {
		if err := p.runStage(ctx, result, StageCollect, p.collect); err != nil {
			return result, err
		}
	}

	// Stage 6: Generate report
	if err := p.runStage(ctx, result, StageReport, p.report); err != nil {
		return result, err
	}

	result.Finalize()

	// Final summary
	p.printFinalSummary(result)

	return result, nil
}

// runStage executes a pipeline stage with timing and error handling
func (p *Pipeline) runStage(ctx context.Context, result *Result, stage Stage, fn func(context.Context) error) error {
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("  Stage %d: %s\n", stage+1, stage.String())
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	sr := &StageResult{
		Stage:    stage,
		Success:  err == nil,
		Duration: duration,
	}

	if err != nil {
		sr.Error = err
		sr.Message = fmt.Sprintf("Failed: %v", err)
		fmt.Printf("\n[FAIL] Stage %s failed: %v\n", stage.String(), err)
	} else {
		sr.Message = fmt.Sprintf("Completed in %s", duration)
		fmt.Printf("\n[OK] Stage %s completed in %s\n", stage.String(), duration)
	}

	result.AddStageResult(sr)
	return err
}

// Stage 1: Initialize
func (p *Pipeline) initialize(ctx context.Context) error {
	fmt.Println("Initializing pipeline...")

	// Get chain ID
	chainID, err := p.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}
	p.chainID = chainID

	if p.cfg.ChainID == 0 {
		p.cfg.ChainID = chainID.Uint64()
	}

	// Display configuration
	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  URL:            %s\n", p.cfg.URL)
	fmt.Printf("  Chain ID:       %d\n", p.cfg.ChainID)
	fmt.Printf("  Mode:           %s\n", p.cfg.Mode)
	fmt.Printf("  Master Account: %s\n", p.wallet.MasterAddress().Hex())
	fmt.Printf("  Sub Accounts:   %d\n", p.cfg.SubAccounts)
	fmt.Printf("  Transactions:   %d\n", p.cfg.Transactions)
	fmt.Printf("  Batch Size:     %d\n", p.cfg.BatchSize)
	fmt.Printf("  Gas Limit:      %d\n", p.cfg.GasLimit)

	// Check master balance
	masterBalance, err := p.client.BalanceAt(ctx, p.wallet.MasterAddress(), nil)
	if err != nil {
		return fmt.Errorf("failed to get master balance: %w", err)
	}
	fmt.Printf("\nMaster Balance: %s wei\n", masterBalance.String())

	// Initialize components
	p.initializeComponents()

	return nil
}

// initializeComponents initializes all pipeline components
func (p *Pipeline) initializeComponents() {
	// Determine gas price for distributor
	distGasPrice := big.NewInt(1000000000) // 1 Gwei default
	if p.cfg.GasPrice != "" {
		if gasPrice, ok := new(big.Int).SetString(p.cfg.GasPrice, 10); ok && gasPrice.Sign() > 0 {
			distGasPrice = gasPrice
		}
	}

	// Distributor
	distCfg := &distributor.Config{
		GasPerTx:      p.cfg.GasLimit,
		TxsPerAccount: int(p.cfg.Transactions / p.cfg.SubAccounts),
		GasPrice:      distGasPrice,
		BufferPercent: 20,
	}
	p.distributor = distributor.New(p.client, distCfg)

	// Batcher - optimized for maximum throughput
	batchCfg := &batcher.Config{
		BatchSize:     int(p.cfg.BatchSize),
		MaxConcurrent: 100, // Increased from 10 for parallel sending
		BatchInterval: 0,   // Removed delay for maximum speed
		RetryCount:    3,
		RetryDelay:    500 * time.Millisecond,
		Timeout:       30 * time.Second,
	}
	p.batcher = batcher.New(p.client, batchCfg)

	// Streamer (if streaming mode)
	if p.runCfg.StreamingMode {
		streamCfg := &batcher.StreamerConfig{
			Rate:    p.runCfg.StreamingRate,
			Burst:   100,
			Workers: 10,
			Timeout: 5 * time.Second,
		}
		p.streamer = batcher.NewStreamer(p.client, streamCfg)
	}

	// Collector
	collCfg := &collector.Config{
		PollInterval:         500 * time.Millisecond,
		ConfirmTimeout:       p.cfg.Timeout,
		MaxConcurrent:        20,
		BatchSize:            100,
		BlockTrackingEnabled: true,
		BlockPollInterval:    1 * time.Second,
	}
	p.collector = collector.New(p.client, collCfg)
}

// Stage 2: Distribute funds
func (p *Pipeline) distribute(ctx context.Context) error {
	fmt.Println("Distributing funds to sub-accounts...")

	subAddrs := p.wallet.SubAddresses()

	result, err := p.distributor.Distribute(ctx, p.wallet.MasterKey(), subAddrs)
	if err != nil {
		return fmt.Errorf("distribution failed: %w", err)
	}

	// Wait for funding to confirm if any transactions were sent
	if result.TxCount > 0 {
		if err := p.distributor.WaitForFunding(ctx, result.ReadyAccounts, 60*time.Second); err != nil {
			return fmt.Errorf("failed waiting for funding: %w", err)
		}
	}

	// Get nonces for building transactions
	p.nonces, err = p.distributor.GetAccountNonces(ctx, result.ReadyAccounts)
	if err != nil {
		return fmt.Errorf("failed to get nonces: %w", err)
	}

	fmt.Printf("\nDistribution Summary:\n")
	fmt.Printf("  Ready Accounts:    %d\n", len(result.ReadyAccounts))
	fmt.Printf("  Unfunded Accounts: %d\n", len(result.UnfundedAccounts))
	fmt.Printf("  Total Distributed: %s wei\n", result.TotalDistributed.String())
	fmt.Printf("  Transactions Sent: %d\n", result.TxCount)

	return nil
}

// Stage 3: Build transactions
func (p *Pipeline) build(ctx context.Context) error {
	fmt.Println("Building transactions...")

	// Create builder config
	builderCfg := &txbuilder.BuilderConfig{
		ChainID:  p.chainID,
		GasLimit: p.cfg.GasLimit,
	}

	// Apply gas price from config if specified
	if p.cfg.GasPrice != "" {
		gasPrice, ok := new(big.Int).SetString(p.cfg.GasPrice, 10)
		if ok && gasPrice.Sign() > 0 {
			builderCfg.GasPrice = gasPrice
			builderCfg.GasTipCap = gasPrice
			builderCfg.GasFeeCap = gasPrice
		}
	}

	// Apply transfer value from config (default: 1 wei)
	if p.cfg.Value != "" {
		value, ok := new(big.Int).SetString(p.cfg.Value, 10)
		if ok && value.Sign() >= 0 {
			builderCfg.Value = value
		}
	}
	if builderCfg.Value == nil {
		builderCfg.Value = big.NewInt(1)
	}

	// Create factory
	factory := txbuilder.NewFactory(builderCfg, p.client)

	// Create builder based on mode
	var err error
	p.builder, err = p.createBuilder(factory)
	if err != nil {
		return fmt.Errorf("failed to create builder: %w", err)
	}

	// Get keys and ensure nonces are set
	keys := p.wallet.SubKeys()
	if len(p.nonces) == 0 {
		p.nonces = make([]uint64, len(keys))
		for i, key := range keys {
			addr := crypto.PubkeyToAddress(key.PublicKey)
			nonce, err := p.client.PendingNonceAt(ctx, addr)
			if err != nil {
				return fmt.Errorf("failed to get nonce for %s: %w", addr.Hex(), err)
			}
			p.nonces[i] = nonce
		}
	}

	// Build transactions
	p.signedTxs, err = p.builder.Build(ctx, keys, p.nonces, int(p.cfg.Transactions))
	if err != nil {
		return fmt.Errorf("failed to build transactions: %w", err)
	}

	fmt.Printf("\nBuild Summary:\n")
	fmt.Printf("  Builder:           %s\n", p.builder.Name())
	fmt.Printf("  Total Built:       %d\n", len(p.signedTxs))

	return nil
}

// createBuilder creates a builder based on the mode
func (p *Pipeline) createBuilder(factory *txbuilder.Factory) (txbuilder.Builder, error) {
	mode := p.cfg.GetMode()
	var opts []txbuilder.BuilderOption

	switch mode {
	case config.ModeTransfer:
		// Self-transfer by default
		return factory.CreateBuilder(mode, opts...)

	case config.ModeFeeDelegation:
		// Parse fee payer key
		feePayerKey, err := p.parseFeePayerKey()
		if err != nil {
			return nil, err
		}
		opts = append(opts, txbuilder.WithFeePayerKey(feePayerKey))
		return factory.CreateBuilder(mode, opts...)

	case config.ModeContractDeploy:
		return factory.CreateBuilder(mode, opts...)

	case config.ModeContractCall:
		contractAddr := common.HexToAddress(p.cfg.Contract)
		opts = append(opts, txbuilder.WithContractAddress(contractAddr))
		opts = append(opts, txbuilder.WithMethod(p.cfg.Method))
		return factory.CreateBuilder(mode, opts...)

	case config.ModeERC20Transfer:
		tokenAddr := common.HexToAddress(p.cfg.Contract)
		opts = append(opts, txbuilder.WithTokenAddress(tokenAddr))
		return factory.CreateBuilder(mode, opts...)

	case config.ModeERC721Mint:
		opts = append(opts, txbuilder.WithNFTName(p.cfg.NFTName))
		opts = append(opts, txbuilder.WithNFTSymbol(p.cfg.NFTSymbol))
		opts = append(opts, txbuilder.WithTokenURI(p.cfg.TokenURI))
		if p.cfg.Contract != "" {
			nftAddr := common.HexToAddress(p.cfg.Contract)
			opts = append(opts, txbuilder.WithNFTContract(nftAddr))
		}
		return factory.CreateBuilder(mode, opts...)

	default:
		return nil, fmt.Errorf("unsupported mode: %s", mode)
	}
}

// parseFeePayerKey parses the fee payer private key
func (p *Pipeline) parseFeePayerKey() (*ecdsa.PrivateKey, error) {
	keyHex := p.cfg.FeePayerKey
	if len(keyHex) >= 2 && keyHex[:2] == "0x" {
		keyHex = keyHex[2:]
	}
	return crypto.HexToECDSA(keyHex)
}

// Stage 4: Send transactions
func (p *Pipeline) send(ctx context.Context) error {
	fmt.Println("Sending transactions...")

	if len(p.signedTxs) == 0 {
		return fmt.Errorf("no transactions to send")
	}

	// Track transactions in collector
	for _, tx := range p.signedTxs {
		p.collector.TrackTransaction(tx.Hash, tx.From, tx.Nonce, tx.GasLimit, time.Now())
	}

	// Send using appropriate method
	if p.runCfg.StreamingMode && p.streamer != nil {
		_, err := p.streamer.Stream(ctx, p.signedTxs)
		return err
	}

	_, err := p.batcher.SendAll(ctx, p.signedTxs)
	return err
}

// Stage 5: Collect results
func (p *Pipeline) collect(ctx context.Context) error {
	fmt.Println("Collecting transaction receipts...")

	report, err := p.collector.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collection failed: %w", err)
	}

	// Store report for later use
	p.collector.Reset()

	// Export if configured
	if p.runCfg.ExportReport && p.runCfg.OutputDir != "" {
		exporter := collector.NewExporter(p.runCfg.OutputDir)
		files, err := exporter.ExportAll(report)
		if err != nil {
			fmt.Printf("[WARN] Failed to export report: %v\n", err)
		} else {
			fmt.Printf("\nReports exported to:\n")
			for _, f := range files {
				fmt.Printf("  - %s\n", f)
			}
		}
	}

	return nil
}

// Stage 6: Generate report
func (p *Pipeline) report(ctx context.Context) error {
	fmt.Println("Generating final report...")
	// Report is already generated in collect stage
	return nil
}

// printFinalSummary prints the final execution summary
func (p *Pipeline) printFinalSummary(result *Result) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      Execution Summary                        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Stage summary
	fmt.Printf("Stage Results:\n")
	for _, sr := range result.StageResults {
		status := "[OK]"
		if !sr.Success {
			status = "[FAIL]"
		}
		fmt.Printf("  %s Stage %d (%s): %s\n", status, sr.Stage+1, sr.Stage.String(), sr.Duration)
	}

	fmt.Printf("\nTotal Duration: %s\n", result.Duration)

	if result.Success() {
		fmt.Println("\nStress test completed successfully!")
	} else {
		fmt.Println("\n[WARN] Stress test completed with errors")
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}
}

// Close cleans up pipeline resources
func (p *Pipeline) Close() {
	if p.client != nil {
		p.client.Close()
	}
}

// executeAnalyzeBlocks runs the block analyzer mode
func (p *Pipeline) executeAnalyzeBlocks(ctx context.Context, result *Result) (*Result, error) {
	fmt.Println("Running Block Analyzer mode...")

	// Create analyzer config
	analyzerCfg := &analyzer.Config{
		StartBlock:  p.cfg.BlockStart,
		EndBlock:    p.cfg.BlockEnd,
		BlockRange:  p.cfg.BlockRange,
		Concurrency: 50,
	}

	// Create and run analyzer
	blockAnalyzer := analyzer.New(p.client, analyzerCfg)

	analysisResult, err := blockAnalyzer.Analyze(ctx)
	if err != nil {
		result.Finalize()
		return result, fmt.Errorf("block analysis failed: %w", err)
	}

	// Print results
	blockAnalyzer.PrintTable(analysisResult)

	// Export to CSV if output directory is configured
	if p.runCfg.OutputDir != "" {
		csvFile := fmt.Sprintf("%s/block_analysis_%d_%d.csv", p.runCfg.OutputDir, analysisResult.StartBlock, analysisResult.EndBlock)
		if err := blockAnalyzer.ExportCSV(analysisResult, csvFile); err != nil {
			fmt.Printf("[WARN] Failed to export CSV: %v\n", err)
		} else {
			fmt.Printf("\nAnalysis exported to: %s\n", csvFile)
		}
	}

	result.Finalize()
	fmt.Println("\nBlock analysis completed successfully!")
	return result, nil
}

// executeLongSender runs the long sender mode
func (p *Pipeline) executeLongSender(ctx context.Context, result *Result, metricsServer *metrics.Metrics) (*Result, error) {
	fmt.Println("Running Long Sender mode...")

	// Get chain ID
	chainID, err := p.client.ChainID(ctx)
	if err != nil {
		result.Finalize()
		return result, fmt.Errorf("failed to get chain ID: %w", err)
	}

	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  URL:            %s\n", p.cfg.URL)
	fmt.Printf("  Chain ID:       %d\n", chainID.Uint64())
	fmt.Printf("  Duration:       %s\n", p.cfg.Duration)
	fmt.Printf("  Target TPS:     %.2f\n", p.cfg.TargetTPS)
	fmt.Printf("  Workers:        %d\n", p.cfg.Workers)
	fmt.Printf("  Accounts:       %d\n", p.cfg.SubAccounts)

	// Get keys and initial nonces
	keys := p.wallet.SubKeys()
	initialNonces := make([]uint64, len(keys))

	for i, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		nonce, err := p.client.PendingNonceAt(ctx, addr)
		if err != nil {
			result.Finalize()
			return result, fmt.Errorf("failed to get nonce for %s: %w", addr.Hex(), err)
		}
		initialNonces[i] = nonce
	}

	// Create monitor
	mon := monitor.New(monitor.DefaultConfig())
	mon.Start()

	// Create long sender config
	senderCfg := &longsender.Config{
		Duration: p.cfg.Duration,
		TPS:      p.cfg.TargetTPS,
		Burst:    int(p.cfg.TargetTPS / 10),
		Workers:  p.cfg.Workers,
	}
	if senderCfg.Burst < 10 {
		senderCfg.Burst = 10
	}

	// Create long sender with callbacks
	sender := longsender.New(p.client, senderCfg)

	// Setup callbacks for metrics and monitoring
	callbacks := &longsender.Callbacks{
		OnSent: func(hash common.Hash) {
			mon.RecordSent(1)
			if metricsServer != nil {
				metricsServer.RecordTxSent()
			}
		},
		OnFailed: func(err error) {
			mon.RecordFailed(1)
			if metricsServer != nil {
				metricsServer.RecordTxFailed()
			}
		},
		OnTPS: func(currentTPS float64) {
			if metricsServer != nil {
				metricsServer.SetCurrentTPS(currentTPS)
			}
		},
	}
	sender.WithCallbacks(callbacks)

	// Start monitor display in background
	monCtx, monCancel := context.WithCancel(ctx)
	go mon.Display(monCtx)

	fmt.Println("\nStarting continuous transaction sending...")
	fmt.Println("Press Ctrl+C to stop")

	// Run the long sender
	sendResult, err := sender.Run(ctx, keys, initialNonces)

	// Stop monitor display
	monCancel()

	// Print final results
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     Long Sender Results                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	if sendResult != nil {
		fmt.Printf("  Total Duration:     %s\n", sendResult.TotalDuration)
		fmt.Printf("  Transactions Sent:  %d\n", sendResult.TotalSent)
		fmt.Printf("  Transactions Failed: %d\n", sendResult.TotalFailed)
		fmt.Printf("  Average TPS:        %.2f\n", sendResult.AverageTPS)
		fmt.Printf("  Success Rate:       %.2f%%\n", float64(sendResult.TotalSent)/float64(sendResult.TotalSent+sendResult.TotalFailed)*100)

		if len(sendResult.Errors) > 0 {
			fmt.Printf("\n  Sample Errors (last %d):\n", len(sendResult.Errors))
			for i, e := range sendResult.Errors {
				if i >= 5 {
					fmt.Printf("    ... and %d more\n", len(sendResult.Errors)-5)
					break
				}
				fmt.Printf("    - %v\n", e)
			}
		}
	}

	result.Finalize()

	if err != nil {
		if ctx.Err() != nil {
			fmt.Println("\nLong sender stopped by user")
			return result, nil
		}
		return result, fmt.Errorf("long sender failed: %w", err)
	}

	fmt.Println("\nLong sender completed successfully!")
	return result, nil
}
