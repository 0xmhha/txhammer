package txbuilder

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xmhha/txhammer/internal/config"
)

// Factory creates builders based on configuration
type Factory struct {
	cfg       *BuilderConfig
	estimator GasEstimator
}

// NewFactory creates a new builder factory
func NewFactory(cfg *BuilderConfig, estimator GasEstimator) *Factory {
	return &Factory{
		cfg:       cfg,
		estimator: estimator,
	}
}

// CreateBuilder creates a builder based on the mode
func (f *Factory) CreateBuilder(mode config.Mode, opts ...BuilderOption) (Builder, error) {
	options := &builderOptions{}
	for _, opt := range opts {
		opt(options)
	}

	return f.buildBuilder(mode, options)
}

func (f *Factory) buildBuilder(mode config.Mode, options *builderOptions) (Builder, error) {
	switch mode {
	case config.ModeTransfer:
		return f.buildTransfer(options), nil
	case config.ModeFeeDelegation:
		return f.buildFeeDelegation(options)
	case config.ModeContractDeploy:
		return f.buildContractDeploy(options), nil
	case config.ModeContractCall:
		return f.buildContractCall(options)
	case config.ModeERC20Transfer:
		return f.buildERC20Transfer(options)
	case config.ModeERC721Mint:
		return f.buildERC721Mint(options)
	case config.ModeLongSender, config.ModeAnalyzeBlocks:
		return nil, fmt.Errorf("mode %s does not use a transaction builder", mode)
	default:
		return nil, fmt.Errorf("unsupported mode: %s", mode)
	}
}

func (f *Factory) buildTransfer(options *builderOptions) *TransferBuilder {
	builder := NewTransferBuilder(f.cfg, f.estimator)
	if options.recipient != (common.Address{}) {
		builder.WithRecipient(options.recipient)
	}
	return builder
}

func (f *Factory) buildFeeDelegation(options *builderOptions) (Builder, error) {
	if options.feePayerKey == nil {
		return nil, fmt.Errorf("fee payer key is required for FEE_DELEGATION mode")
	}
	builder := NewFeeDelegationBuilder(f.cfg, f.estimator, options.feePayerKey)
	if options.recipient != (common.Address{}) {
		builder.WithRecipient(options.recipient)
	}
	return builder, nil
}

func (f *Factory) buildContractDeploy(options *builderOptions) *ContractDeployBuilder {
	builder := NewContractDeployBuilder(f.cfg, f.estimator)
	if options.bytecode != nil {
		builder.WithBytecode(options.bytecode)
	}
	return builder
}

func (f *Factory) buildContractCall(options *builderOptions) (Builder, error) {
	if options.contractAddr == (common.Address{}) {
		return nil, fmt.Errorf("contract address is required for CONTRACT_CALL mode")
	}
	builder := NewContractCallBuilder(f.cfg, f.estimator, options.contractAddr)
	if options.method != "" {
		builder.WithMethod(options.method, options.methodArgs...)
	}
	if options.abiJSON != "" {
		var err error
		builder, err = builder.WithABI(options.abiJSON)
		if err != nil {
			return nil, err
		}
	}
	return builder, nil
}

func (f *Factory) buildERC20Transfer(options *builderOptions) (Builder, error) {
	if options.tokenAddr == (common.Address{}) {
		return nil, fmt.Errorf("token address is required for ERC20_TRANSFER mode")
	}
	builder := NewERC20TransferBuilder(f.cfg, f.estimator, options.tokenAddr)
	if options.recipient != (common.Address{}) {
		builder.WithRecipient(options.recipient)
	}
	if options.amount != nil {
		builder.WithAmount(options.amount)
	}
	return builder, nil
}

func (f *Factory) buildERC721Mint(options *builderOptions) (Builder, error) {
	builder, err := NewERC721MintBuilder(f.cfg, f.estimator)
	if err != nil {
		return nil, fmt.Errorf("failed to create ERC721 mint builder: %w", err)
	}
	if options.nftContract != (common.Address{}) {
		builder.WithContract(options.nftContract)
	}
	if options.tokenURI != "" {
		builder.WithTokenURI(options.tokenURI)
	}
	if options.nftName != "" {
		builder.WithNFTName(options.nftName)
	}
	if options.nftSymbol != "" {
		builder.WithNFTSymbol(options.nftSymbol)
	}
	return builder, nil
}

// BuilderOption is a functional option for builder configuration
type BuilderOption func(*builderOptions)

type builderOptions struct {
	recipient    common.Address
	feePayerKey  *ecdsa.PrivateKey
	contractAddr common.Address
	tokenAddr    common.Address
	bytecode     []byte
	method       string
	methodArgs   []interface{}
	abiJSON      string
	amount       *big.Int
	// ERC721 options
	nftContract common.Address
	tokenURI    string
	nftName     string
	nftSymbol   string
}

// WithRecipient sets the recipient address
func WithRecipient(addr common.Address) BuilderOption {
	return func(o *builderOptions) {
		o.recipient = addr
	}
}

// WithFeePayerKey sets the fee payer key for fee delegation
func WithFeePayerKey(key *ecdsa.PrivateKey) BuilderOption {
	return func(o *builderOptions) {
		o.feePayerKey = key
	}
}

// WithContractAddress sets the contract address
func WithContractAddress(addr common.Address) BuilderOption {
	return func(o *builderOptions) {
		o.contractAddr = addr
	}
}

// WithTokenAddress sets the token address for ERC20
func WithTokenAddress(addr common.Address) BuilderOption {
	return func(o *builderOptions) {
		o.tokenAddr = addr
	}
}

// WithBytecode sets the contract bytecode
func WithBytecode(bytecode []byte) BuilderOption {
	return func(o *builderOptions) {
		o.bytecode = bytecode
	}
}

// WithMethod sets the contract method
func WithMethod(method string, args ...interface{}) BuilderOption {
	return func(o *builderOptions) {
		o.method = method
		o.methodArgs = args
	}
}

// WithABI sets the contract ABI
func WithABI(abiJSON string) BuilderOption {
	return func(o *builderOptions) {
		o.abiJSON = abiJSON
	}
}

// WithAmount sets the transfer amount
func WithAmount(amount *big.Int) BuilderOption {
	return func(o *builderOptions) {
		o.amount = amount
	}
}

// WithNFTContract sets the NFT contract address
func WithNFTContract(addr common.Address) BuilderOption {
	return func(o *builderOptions) {
		o.nftContract = addr
	}
}

// WithTokenURI sets the token URI for NFT minting
func WithTokenURI(uri string) BuilderOption {
	return func(o *builderOptions) {
		o.tokenURI = uri
	}
}

// WithNFTName sets the NFT collection name
func WithNFTName(name string) BuilderOption {
	return func(o *builderOptions) {
		o.nftName = name
	}
}

// WithNFTSymbol sets the NFT collection symbol
func WithNFTSymbol(symbol string) BuilderOption {
	return func(o *builderOptions) {
		o.nftSymbol = symbol
	}
}
