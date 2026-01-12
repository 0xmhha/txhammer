package wallet

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	testPrivateKey = "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testMnemonic   = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
)

func TestNewFromPrivateKey(t *testing.T) {
	tests := []struct {
		name        string
		privateKey  string
		subAccounts uint64
		wantErr     bool
	}{
		{
			name:        "valid key with 0x prefix",
			privateKey:  testPrivateKey,
			subAccounts: 5,
			wantErr:     false,
		},
		{
			name:        "valid key without 0x prefix",
			privateKey:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			subAccounts: 5,
			wantErr:     false,
		},
		{
			name:        "invalid key",
			privateKey:  "invalid",
			subAccounts: 5,
			wantErr:     true,
		},
		{
			name:        "zero sub accounts",
			privateKey:  testPrivateKey,
			subAccounts: 0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := NewFromPrivateKey(tt.privateKey, tt.subAccounts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFromPrivateKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if w.MasterKey() == nil {
					t.Error("MasterKey() returned nil")
				}
				if len(w.SubKeys()) != int(tt.subAccounts) {
					t.Errorf("SubKeys() count = %d, want %d", len(w.SubKeys()), tt.subAccounts)
				}
			}
		})
	}
}

func TestNewFromMnemonic(t *testing.T) {
	tests := []struct {
		name        string
		mnemonic    string
		subAccounts uint64
		wantErr     bool
	}{
		{
			name:        "valid mnemonic",
			mnemonic:    testMnemonic,
			subAccounts: 5,
			wantErr:     false,
		},
		{
			name:        "invalid mnemonic",
			mnemonic:    "invalid mnemonic phrase",
			subAccounts: 5,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := NewFromMnemonic(tt.mnemonic, tt.subAccounts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFromMnemonic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if w.MasterKey() == nil {
					t.Error("MasterKey() returned nil")
				}
				if len(w.SubKeys()) != int(tt.subAccounts) {
					t.Errorf("SubKeys() count = %d, want %d", len(w.SubKeys()), tt.subAccounts)
				}
			}
		})
	}
}

func TestWallet_MasterAddress(t *testing.T) {
	w, err := NewFromPrivateKey(testPrivateKey, 5)
	if err != nil {
		t.Fatalf("NewFromPrivateKey() failed: %v", err)
	}

	addr := w.MasterAddress()
	if addr == (common.Address{}) {
		t.Error("MasterAddress() returned zero address")
	}

	// Verify address matches derived from public key
	expectedAddr := crypto.PubkeyToAddress(w.MasterKey().PublicKey)
	if addr != expectedAddr {
		t.Errorf("MasterAddress() = %s, want %s", addr.Hex(), expectedAddr.Hex())
	}
}

func TestWallet_SubAddresses(t *testing.T) {
	subAccounts := uint64(5)
	w, err := NewFromPrivateKey(testPrivateKey, subAccounts)
	if err != nil {
		t.Fatalf("NewFromPrivateKey() failed: %v", err)
	}

	addrs := w.SubAddresses()
	if len(addrs) != int(subAccounts) {
		t.Errorf("SubAddresses() count = %d, want %d", len(addrs), subAccounts)
	}

	// Verify all addresses are unique
	seen := make(map[common.Address]bool)
	for i, addr := range addrs {
		if addr == (common.Address{}) {
			t.Errorf("SubAddresses()[%d] is zero address", i)
		}
		if seen[addr] {
			t.Errorf("SubAddresses()[%d] is duplicate", i)
		}
		seen[addr] = true
	}
}

func TestWallet_AllKeys(t *testing.T) {
	subAccounts := uint64(5)
	w, err := NewFromPrivateKey(testPrivateKey, subAccounts)
	if err != nil {
		t.Fatalf("NewFromPrivateKey() failed: %v", err)
	}

	keys := w.AllKeys()
	expectedCount := 1 + int(subAccounts) // master + sub accounts
	if len(keys) != expectedCount {
		t.Errorf("AllKeys() count = %d, want %d", len(keys), expectedCount)
	}

	// First key should be master key
	if keys[0] != w.MasterKey() {
		t.Error("AllKeys()[0] is not master key")
	}
}

func TestWallet_AllAddresses(t *testing.T) {
	subAccounts := uint64(5)
	w, err := NewFromPrivateKey(testPrivateKey, subAccounts)
	if err != nil {
		t.Fatalf("NewFromPrivateKey() failed: %v", err)
	}

	addrs := w.AllAddresses()
	expectedCount := 1 + int(subAccounts)
	if len(addrs) != expectedCount {
		t.Errorf("AllAddresses() count = %d, want %d", len(addrs), expectedCount)
	}

	// First address should be master address
	if addrs[0] != w.MasterAddress() {
		t.Error("AllAddresses()[0] is not master address")
	}
}

func TestWallet_DeterministicDerivation(t *testing.T) {
	// Create two wallets from same key
	w1, err := NewFromPrivateKey(testPrivateKey, 5)
	if err != nil {
		t.Fatalf("NewFromPrivateKey() failed: %v", err)
	}

	w2, err := NewFromPrivateKey(testPrivateKey, 5)
	if err != nil {
		t.Fatalf("NewFromPrivateKey() failed: %v", err)
	}

	// Master addresses should match
	if w1.MasterAddress() != w2.MasterAddress() {
		t.Error("Master addresses should be deterministic")
	}

	// Sub addresses should match
	addrs1 := w1.SubAddresses()
	addrs2 := w2.SubAddresses()
	for i := range addrs1 {
		if addrs1[i] != addrs2[i] {
			t.Errorf("SubAddresses()[%d] not deterministic", i)
		}
	}
}

func TestWallet_MnemonicDeterministicDerivation(t *testing.T) {
	// Create two wallets from same mnemonic
	w1, err := NewFromMnemonic(testMnemonic, 5)
	if err != nil {
		t.Fatalf("NewFromMnemonic() failed: %v", err)
	}

	w2, err := NewFromMnemonic(testMnemonic, 5)
	if err != nil {
		t.Fatalf("NewFromMnemonic() failed: %v", err)
	}

	// Master addresses should match
	if w1.MasterAddress() != w2.MasterAddress() {
		t.Error("Master addresses should be deterministic")
	}

	// Sub addresses should match
	addrs1 := w1.SubAddresses()
	addrs2 := w2.SubAddresses()
	for i := range addrs1 {
		if addrs1[i] != addrs2[i] {
			t.Errorf("SubAddresses()[%d] not deterministic", i)
		}
	}
}

func TestSignHash(t *testing.T) {
	w, err := NewFromPrivateKey(testPrivateKey, 0)
	if err != nil {
		t.Fatalf("NewFromPrivateKey() failed: %v", err)
	}

	hash := crypto.Keccak256([]byte("test message"))
	sig, err := SignHash(w.MasterKey(), hash)
	if err != nil {
		t.Fatalf("SignHash() failed: %v", err)
	}

	if len(sig) != 65 {
		t.Errorf("Signature length = %d, want 65", len(sig))
	}

	// Verify signature can recover public key
	pubKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		t.Fatalf("SigToPub() failed: %v", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	if recoveredAddr != w.MasterAddress() {
		t.Errorf("Recovered address = %s, want %s", recoveredAddr.Hex(), w.MasterAddress().Hex())
	}
}

func TestAddressFromPrivateKey(t *testing.T) {
	w, err := NewFromPrivateKey(testPrivateKey, 0)
	if err != nil {
		t.Fatalf("NewFromPrivateKey() failed: %v", err)
	}

	addr := AddressFromPrivateKey(w.MasterKey())
	if addr != w.MasterAddress() {
		t.Errorf("AddressFromPrivateKey() = %s, want %s", addr.Hex(), w.MasterAddress().Hex())
	}
}

func TestWallet_UniqueSubAccounts(t *testing.T) {
	w, err := NewFromPrivateKey(testPrivateKey, 100)
	if err != nil {
		t.Fatalf("NewFromPrivateKey() failed: %v", err)
	}

	// Check all sub accounts are unique
	seen := make(map[common.Address]bool)
	for i, addr := range w.SubAddresses() {
		if seen[addr] {
			t.Errorf("Duplicate address at index %d: %s", i, addr.Hex())
		}
		seen[addr] = true
	}

	// Master should not be in sub accounts
	if seen[w.MasterAddress()] {
		t.Error("Master address found in sub accounts")
	}
}
