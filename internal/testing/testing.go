// Package testing provides test utilities and helpers for txhammer tests.
package testing

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// TestPrivateKey is a well-known test private key (DO NOT use in production)
const TestPrivateKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// TestMnemonic is a well-known test mnemonic (DO NOT use in production)
const TestMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

// TestChainID is the default chain ID for tests
var TestChainID = big.NewInt(1337)

// GenerateTestKey generates a random private key for testing
func GenerateTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	return key
}

// GenerateTestKeys generates multiple random private keys for testing
func GenerateTestKeys(t *testing.T, count int) []*ecdsa.PrivateKey {
	t.Helper()
	keys := make([]*ecdsa.PrivateKey, count)
	for i := range count {
		keys[i] = GenerateTestKey(t)
	}
	return keys
}

// MustParseKey parses a hex private key or fails the test
func MustParseKey(t *testing.T, hexKey string) *ecdsa.PrivateKey {
	t.Helper()
	if len(hexKey) >= 2 && hexKey[:2] == "0x" {
		hexKey = hexKey[2:]
	}
	key, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		t.Fatalf("failed to parse private key: %v", err)
	}
	return key
}

// AddressFromKey returns the address for a private key
func AddressFromKey(key *ecdsa.PrivateKey) common.Address {
	return crypto.PubkeyToAddress(key.PublicKey)
}

// RandomAddress generates a random address for testing
func RandomAddress(t *testing.T) common.Address {
	t.Helper()
	key := GenerateTestKey(t)
	return AddressFromKey(key)
}

// RandomAddresses generates multiple random addresses for testing
func RandomAddresses(t *testing.T, count int) []common.Address {
	t.Helper()
	addrs := make([]common.Address, count)
	for i := range count {
		addrs[i] = RandomAddress(t)
	}
	return addrs
}

// HexToAddress converts a hex string to an address
func HexToAddress(hex string) common.Address {
	return common.HexToAddress(hex)
}

// HexToHash converts a hex string to a hash
func HexToHash(hex string) common.Hash {
	return common.HexToHash(hex)
}

// BigInt creates a big.Int from an int64
func BigInt(n int64) *big.Int {
	return big.NewInt(n)
}

// Ether converts ether to wei
func Ether(n int64) *big.Int {
	wei := big.NewInt(n)
	return wei.Mul(wei, big.NewInt(1e18))
}

// Gwei converts gwei to wei
func Gwei(n int64) *big.Int {
	wei := big.NewInt(n)
	return wei.Mul(wei, big.NewInt(1e9))
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

// AssertEqual fails the test if got != want
func AssertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// AssertNotEqual fails the test if got == want
func AssertNotEqual[T comparable](t *testing.T, got, notWant T) {
	t.Helper()
	if got == notWant {
		t.Errorf("got %v, should not equal %v", got, notWant)
	}
}

// AssertTrue fails the test if condition is false
func AssertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Errorf("assertion failed: %s", msg)
	}
}

// AssertFalse fails the test if condition is true
func AssertFalse(t *testing.T, condition bool, msg string) {
	t.Helper()
	if condition {
		t.Errorf("assertion failed (expected false): %s", msg)
	}
}

// AssertNil fails the test if value is not nil
func AssertNil(t *testing.T, value any) {
	t.Helper()
	if value != nil {
		t.Errorf("expected nil, got %v", value)
	}
}

// AssertNotNil fails the test if value is nil
func AssertNotNil(t *testing.T, value any) {
	t.Helper()
	if value == nil {
		t.Error("expected non-nil value")
	}
}

// AssertLen fails the test if the slice length doesn't match
func AssertLen[T any](t *testing.T, slice []T, expectedLen int) {
	t.Helper()
	if len(slice) != expectedLen {
		t.Errorf("expected length %d, got %d", expectedLen, len(slice))
	}
}
