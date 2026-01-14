package txbuilder

import (
	"context"
	"crypto/ecdsa"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/schollz/progressbar/v3"

	"github.com/0xmhha/txhammer/internal/util/progress"
)

//go:embed contracts/ZexNFTs.json
var zexNFTsJSON []byte

// ContractArtifact represents the compiled contract JSON structure
type ContractArtifact struct {
	ContractName string          `json:"contractName"`
	ABI          json.RawMessage `json:"abi"`
	Bytecode     string          `json:"bytecode"`
}

// ERC721MintBuilder builds ERC721 NFT minting transactions
type ERC721MintBuilder struct {
	*BaseBuilder
	nftContract    common.Address
	tokenURI       string
	nftName        string
	nftSymbol      string
	contractABI    abi.ABI
	deployBytecode []byte
}

// NewERC721MintBuilder creates a new ERC721 mint builder
func NewERC721MintBuilder(config *BuilderConfig, estimator GasEstimator) (*ERC721MintBuilder, error) {
	// Parse the embedded contract artifact
	var artifact ContractArtifact
	if err := json.Unmarshal(zexNFTsJSON, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse contract artifact: %w", err)
	}

	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Parse the bytecode
	bytecode := common.FromHex(artifact.Bytecode)

	return &ERC721MintBuilder{
		BaseBuilder:    NewBaseBuilder(config, estimator),
		contractABI:    parsedABI,
		deployBytecode: bytecode,
		nftName:        "TxHammerNFT",
		nftSymbol:      "TXHNFT",
		tokenURI:       "https://txhammer.io/nft/",
	}, nil
}

// WithContract sets the NFT contract address
func (b *ERC721MintBuilder) WithContract(addr common.Address) *ERC721MintBuilder {
	b.nftContract = addr
	return b
}

// WithTokenURI sets the base token URI for minting
func (b *ERC721MintBuilder) WithTokenURI(uri string) *ERC721MintBuilder {
	b.tokenURI = uri
	return b
}

// WithNFTName sets the NFT collection name (for deployment)
func (b *ERC721MintBuilder) WithNFTName(name string) *ERC721MintBuilder {
	b.nftName = name
	return b
}

// WithNFTSymbol sets the NFT collection symbol (for deployment)
func (b *ERC721MintBuilder) WithNFTSymbol(symbol string) *ERC721MintBuilder {
	b.nftSymbol = symbol
	return b
}

// Name returns the builder name
func (b *ERC721MintBuilder) Name() string {
	return "ERC721_MINT"
}

// EstimateGas estimates gas for NFT minting
func (b *ERC721MintBuilder) EstimateGas(_ context.Context) (uint64, error) {
	// NFT minting typically needs more gas than simple transfer
	return 150000, nil
}

// GetContractAddress returns the NFT contract address
func (b *ERC721MintBuilder) GetContractAddress() common.Address {
	return b.nftContract
}

// DeployContract deploys the NFT contract and returns the contract address
func (b *ERC721MintBuilder) DeployContract(ctx context.Context, key *ecdsa.PrivateKey, nonce uint64) (common.Address, common.Hash, error) {
	// Pack constructor arguments
	constructorArgs, err := b.contractABI.Pack("", b.nftName, b.nftSymbol)
	if err != nil {
		return common.Address{}, common.Hash{}, fmt.Errorf("failed to pack constructor arguments: %w", err)
	}

	// Combine bytecode with constructor arguments
	deployData := make([]byte, len(b.deployBytecode))
	copy(deployData, b.deployBytecode)
	deployData = append(deployData, constructorArgs...)

	gasTipCap, gasFeeCap, err := b.GetGasSettings(ctx)
	if err != nil {
		return common.Address{}, common.Hash{}, err
	}

	// Contract deployment needs more gas
	gasLimit := uint64(2000000)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   b.config.ChainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        nil, // Contract creation
		Value:     big.NewInt(0),
		Data:      deployData,
	})

	signedTx, err := SignTransaction(tx, b.config.ChainID, key)
	if err != nil {
		return common.Address{}, common.Hash{}, fmt.Errorf("failed to sign deployment transaction: %w", err)
	}

	// Calculate contract address
	from := crypto.PubkeyToAddress(key.PublicKey)
	contractAddr := crypto.CreateAddress(from, nonce)

	return contractAddr, signedTx.Hash(), nil
}

// GetDeployTransaction returns the signed deployment transaction
func (b *ERC721MintBuilder) GetDeployTransaction(ctx context.Context, key *ecdsa.PrivateKey, nonce uint64) (*SignedTx, error) {
	// Pack constructor arguments
	constructorArgs, err := b.contractABI.Pack("", b.nftName, b.nftSymbol)
	if err != nil {
		return nil, fmt.Errorf("failed to pack constructor arguments: %w", err)
	}

	// Combine bytecode with constructor arguments
	deployData := make([]byte, len(b.deployBytecode))
	copy(deployData, b.deployBytecode)
	deployData = append(deployData, constructorArgs...)

	gasTipCap, gasFeeCap, err := b.GetGasSettings(ctx)
	if err != nil {
		return nil, err
	}

	// Contract deployment needs more gas
	gasLimit := uint64(2000000)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   b.config.ChainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        nil, // Contract creation
		Value:     big.NewInt(0),
		Data:      deployData,
	})

	signedTx, err := SignTransaction(tx, b.config.ChainID, key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign deployment transaction: %w", err)
	}

	rawTx, err := signedTx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	from := crypto.PubkeyToAddress(key.PublicKey)

	return &SignedTx{
		Tx:       signedTx,
		RawTx:    rawTx,
		Hash:     signedTx.Hash(),
		From:     from,
		Nonce:    nonce,
		GasLimit: gasLimit,
	}, nil
}

// Build creates ERC721 mint transactions (createNFT calls)
func (b *ERC721MintBuilder) Build(ctx context.Context, keys []*ecdsa.PrivateKey, nonces []uint64, count int) ([]*SignedTx, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys provided")
	}
	if len(keys) != len(nonces) {
		return nil, fmt.Errorf("keys and nonces length mismatch")
	}
	if b.nftContract == (common.Address{}) {
		return nil, fmt.Errorf("NFT contract address is required")
	}

	gasTipCap, gasFeeCap, err := b.GetGasSettings(ctx)
	if err != nil {
		return nil, err
	}

	gasLimit := b.config.GasLimit
	if gasLimit == 0 {
		gasLimit = 150000
	}

	distribution := DistributeTransactions(len(keys), count)

	totalTxs := 0
	for _, n := range distribution {
		totalTxs += n
	}

	fmt.Printf("\nBuilding ERC721 Mint Transactions\n\n")
	fmt.Printf("NFT Contract: %s\n", b.nftContract.Hex())
	fmt.Printf("Token URI Base: %s\n", b.tokenURI)
	bar := progressbar.Default(int64(totalTxs), "txs built")

	signedTxs := make([]*SignedTx, 0, totalTxs)
	tokenID := uint64(0)

	for accountIdx, txCount := range distribution {
		key := keys[accountIdx]
		nonce := nonces[accountIdx]
		from := crypto.PubkeyToAddress(key.PublicKey)

		for i := 0; i < txCount; i++ {
			// Build createNFT call data with unique token URI
			tokenURIWithID := fmt.Sprintf("%s%d", b.tokenURI, tokenID)
			callData, err := b.contractABI.Pack("createNFT", tokenURIWithID)
			if err != nil {
				return nil, fmt.Errorf("failed to pack createNFT call: %w", err)
			}

			tx := types.NewTx(&types.DynamicFeeTx{
				ChainID:   b.config.ChainID,
				Nonce:     nonce,
				GasTipCap: gasTipCap,
				GasFeeCap: gasFeeCap,
				Gas:       gasLimit,
				To:        &b.nftContract,
				Value:     big.NewInt(0),
				Data:      callData,
			})

			signedTx, err := SignTransaction(tx, b.config.ChainID, key)
			if err != nil {
				return nil, fmt.Errorf("failed to sign transaction: %w", err)
			}

			rawTx, err := signedTx.MarshalBinary()
			if err != nil {
				return nil, fmt.Errorf("failed to marshal transaction: %w", err)
			}

			signedTxs = append(signedTxs, &SignedTx{
				Tx:       signedTx,
				RawTx:    rawTx,
				Hash:     signedTx.Hash(),
				From:     from,
				Nonce:    nonce,
				GasLimit: gasLimit,
			})

			nonce++
			tokenID++
			progress.Add(bar, 1)
		}
	}

	fmt.Printf("\n[OK] Successfully built %d ERC721 mint transactions\n", len(signedTxs))
	return signedTxs, nil
}
