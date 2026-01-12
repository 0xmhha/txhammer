package testing

import (
	"math/big"
	"testing"
)

func TestGenerateTestKey(t *testing.T) {
	key := GenerateTestKey(t)
	if key == nil {
		t.Fatal("GenerateTestKey returned nil")
	}
	if key.D == nil {
		t.Error("Private key D is nil")
	}
}

func TestGenerateTestKeys(t *testing.T) {
	keys := GenerateTestKeys(t, 5)
	if len(keys) != 5 {
		t.Errorf("Expected 5 keys, got %d", len(keys))
	}

	// Verify all keys are unique
	seen := make(map[string]bool)
	for i, key := range keys {
		addr := AddressFromKey(key).Hex()
		if seen[addr] {
			t.Errorf("Key %d has duplicate address", i)
		}
		seen[addr] = true
	}
}

func TestMustParseKey(t *testing.T) {
	// With 0x prefix
	key1 := MustParseKey(t, "0x"+TestPrivateKey)
	if key1 == nil {
		t.Fatal("MustParseKey returned nil for prefixed key")
	}

	// Without 0x prefix
	key2 := MustParseKey(t, TestPrivateKey)
	if key2 == nil {
		t.Fatal("MustParseKey returned nil for non-prefixed key")
	}

	// Both should produce the same address
	addr1 := AddressFromKey(key1)
	addr2 := AddressFromKey(key2)
	if addr1 != addr2 {
		t.Errorf("Keys should produce same address: %s vs %s", addr1.Hex(), addr2.Hex())
	}
}

func TestAddressFromKey(t *testing.T) {
	key := GenerateTestKey(t)
	addr := AddressFromKey(key)

	// Address should be 20 bytes
	if len(addr) != 20 {
		t.Errorf("Address should be 20 bytes, got %d", len(addr))
	}

	// Address should be deterministic
	addr2 := AddressFromKey(key)
	if addr != addr2 {
		t.Error("Same key should produce same address")
	}
}

func TestRandomAddress(t *testing.T) {
	addr1 := RandomAddress(t)
	addr2 := RandomAddress(t)

	if addr1 == addr2 {
		t.Error("Random addresses should be different")
	}
}

func TestRandomAddresses(t *testing.T) {
	addrs := RandomAddresses(t, 10)
	if len(addrs) != 10 {
		t.Errorf("Expected 10 addresses, got %d", len(addrs))
	}

	// Check uniqueness
	seen := make(map[string]bool)
	for i, addr := range addrs {
		hex := addr.Hex()
		if seen[hex] {
			t.Errorf("Address %d is duplicate", i)
		}
		seen[hex] = true
	}
}

func TestHexToAddress(t *testing.T) {
	addr := HexToAddress("0x1234567890123456789012345678901234567890")
	expected := "0x1234567890123456789012345678901234567890"
	if addr.Hex() != expected {
		t.Errorf("Expected %s, got %s", expected, addr.Hex())
	}
}

func TestHexToHash(t *testing.T) {
	hash := HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")
	if hash.Hex() != "0x1234567890123456789012345678901234567890123456789012345678901234" {
		t.Errorf("Hash mismatch: %s", hash.Hex())
	}
}

func TestBigInt(t *testing.T) {
	n := BigInt(12345)
	if n.Int64() != 12345 {
		t.Errorf("Expected 12345, got %d", n.Int64())
	}
}

func TestEther(t *testing.T) {
	wei := Ether(1)
	expected := big.NewInt(1e18)
	if wei.Cmp(expected) != 0 {
		t.Errorf("1 ether should be 1e18 wei, got %s", wei.String())
	}

	wei2 := Ether(5)
	expected2 := new(big.Int).Mul(big.NewInt(5), big.NewInt(1e18))
	if wei2.Cmp(expected2) != 0 {
		t.Errorf("5 ether mismatch: %s", wei2.String())
	}
}

func TestGwei(t *testing.T) {
	wei := Gwei(1)
	expected := big.NewInt(1e9)
	if wei.Cmp(expected) != 0 {
		t.Errorf("1 gwei should be 1e9 wei, got %s", wei.String())
	}

	wei2 := Gwei(100)
	expected2 := big.NewInt(100e9)
	if wei2.Cmp(expected2) != 0 {
		t.Errorf("100 gwei mismatch: %s", wei2.String())
	}
}

func TestAssertEqual(t *testing.T) {
	// These should not fail
	AssertEqual(t, 1, 1)
	AssertEqual(t, "hello", "hello")
	AssertEqual(t, true, true)
}

func TestAssertNotEqual(t *testing.T) {
	// These should not fail
	AssertNotEqual(t, 1, 2)
	AssertNotEqual(t, "hello", "world")
	AssertNotEqual(t, true, false)
}

func TestAssertTrue(t *testing.T) {
	AssertTrue(t, true, "should be true")
	AssertTrue(t, 1+1 == 2, "1+1 should equal 2")
}

func TestAssertFalse(t *testing.T) {
	AssertFalse(t, false, "should be false")
	AssertFalse(t, 1 == 2, "1 should not equal 2")
}

func TestAssertLen(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}
	AssertLen(t, slice, 5)

	empty := []string{}
	AssertLen(t, empty, 0)
}

func TestTestChainID(t *testing.T) {
	if TestChainID == nil {
		t.Fatal("TestChainID should not be nil")
	}
	if TestChainID.Int64() != 1337 {
		t.Errorf("TestChainID should be 1337, got %d", TestChainID.Int64())
	}
}

func TestTestMnemonic(t *testing.T) {
	if TestMnemonic == "" {
		t.Fatal("TestMnemonic should not be empty")
	}
	// Should have 12 words
	words := 0
	for _, c := range TestMnemonic {
		if c == ' ' {
			words++
		}
	}
	words++ // Last word doesn't have trailing space
	if words != 12 {
		t.Errorf("TestMnemonic should have 12 words, got %d", words)
	}
}
