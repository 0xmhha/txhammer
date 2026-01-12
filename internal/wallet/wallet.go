package wallet

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

// Wallet manages accounts for stress testing
type Wallet struct {
	masterKey  *ecdsa.PrivateKey
	subKeys    []*ecdsa.PrivateKey
	hdWallet   *hdwallet.Wallet
	useMnemonic bool
}

// NewFromPrivateKey creates a wallet from a private key hex string
func NewFromPrivateKey(privateKeyHex string, subAccounts uint64) (*Wallet, error) {
	// Remove 0x prefix if present
	if len(privateKeyHex) >= 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	masterKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	// Generate sub-accounts by deriving from master key
	subKeys := make([]*ecdsa.PrivateKey, subAccounts)
	for i := uint64(0); i < subAccounts; i++ {
		// Use master key hash + index to derive sub-keys
		seed := crypto.Keccak256(
			crypto.FromECDSA(masterKey),
			[]byte(fmt.Sprintf("subaccount-%d", i)),
		)
		subKey, err := crypto.ToECDSA(seed)
		if err != nil {
			return nil, fmt.Errorf("failed to derive sub-account %d: %w", i, err)
		}
		subKeys[i] = subKey
	}

	return &Wallet{
		masterKey:   masterKey,
		subKeys:     subKeys,
		useMnemonic: false,
	}, nil
}

// NewFromMnemonic creates a wallet from a BIP39 mnemonic
func NewFromMnemonic(mnemonic string, subAccounts uint64) (*Wallet, error) {
	wallet, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		return nil, fmt.Errorf("invalid mnemonic: %w", err)
	}

	// Derive master account (index 0)
	masterPath := hdwallet.MustParseDerivationPath("m/44'/60'/0'/0/0")
	masterAccount, err := wallet.Derive(masterPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to derive master account: %w", err)
	}

	masterKey, err := wallet.PrivateKey(masterAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to get master private key: %w", err)
	}

	// Derive sub-accounts
	subKeys := make([]*ecdsa.PrivateKey, subAccounts)
	for i := uint64(0); i < subAccounts; i++ {
		path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%d", i+1))
		account, err := wallet.Derive(path, false)
		if err != nil {
			return nil, fmt.Errorf("failed to derive sub-account %d: %w", i, err)
		}

		subKey, err := wallet.PrivateKey(account)
		if err != nil {
			return nil, fmt.Errorf("failed to get sub-account %d private key: %w", i, err)
		}
		subKeys[i] = subKey
	}

	return &Wallet{
		masterKey:   masterKey,
		subKeys:     subKeys,
		hdWallet:    wallet,
		useMnemonic: true,
	}, nil
}

// MasterKey returns the master private key
func (w *Wallet) MasterKey() *ecdsa.PrivateKey {
	return w.masterKey
}

// MasterAddress returns the master account address
func (w *Wallet) MasterAddress() common.Address {
	return crypto.PubkeyToAddress(w.masterKey.PublicKey)
}

// SubKeys returns all sub-account private keys
func (w *Wallet) SubKeys() []*ecdsa.PrivateKey {
	return w.subKeys
}

// SubAddresses returns all sub-account addresses
func (w *Wallet) SubAddresses() []common.Address {
	addresses := make([]common.Address, len(w.subKeys))
	for i, key := range w.subKeys {
		addresses[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	return addresses
}

// AllKeys returns all keys (master + sub-accounts)
func (w *Wallet) AllKeys() []*ecdsa.PrivateKey {
	keys := make([]*ecdsa.PrivateKey, 1+len(w.subKeys))
	keys[0] = w.masterKey
	copy(keys[1:], w.subKeys)
	return keys
}

// AllAddresses returns all addresses (master + sub-accounts)
func (w *Wallet) AllAddresses() []common.Address {
	addresses := make([]common.Address, 1+len(w.subKeys))
	addresses[0] = w.MasterAddress()
	copy(addresses[1:], w.SubAddresses())
	return addresses
}

// GetAccount returns an account by index (0 = master, 1+ = sub-accounts)
func (w *Wallet) GetAccount(index int) (accounts.Account, error) {
	if w.hdWallet == nil {
		return accounts.Account{}, fmt.Errorf("account retrieval only available for mnemonic-based wallets")
	}

	path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%d", index))
	return w.hdWallet.Derive(path, false)
}

// SignHash signs a hash with the specified key
func SignHash(key *ecdsa.PrivateKey, hash []byte) ([]byte, error) {
	return crypto.Sign(hash, key)
}

// AddressFromPrivateKey returns the address for a private key
func AddressFromPrivateKey(key *ecdsa.PrivateKey) common.Address {
	return crypto.PubkeyToAddress(key.PublicKey)
}
