package main

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestPKCS7RoundTrip(t *testing.T) {
	input := []byte("device-1546c952")
	padded := pkcs7Pad(input, aesBlockSize)

	if len(padded)%aesBlockSize != 0 {
		t.Fatalf("padded length = %d, want multiple of %d", len(padded), aesBlockSize)
	}

	unpadded, err := pkcs7Unpad(padded, aesBlockSize)
	if err != nil {
		t.Fatalf("pkcs7Unpad returned error: %v", err)
	}
	if !bytes.Equal(unpadded, input) {
		t.Fatalf("unpadded = %q, want %q", unpadded, input)
	}
}

func TestDeriveKeyUsesSHA256(t *testing.T) {
	key := deriveKey("python3806250511")

	if len(key) != 32 {
		t.Fatalf("key length = %d, want 32", len(key))
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	cfg := CryptoConfig{
		Seed: "python3806250511",
		IV:   "0625051106250511",
	}

	encrypted, err := encryptString("1546c952", cfg)
	if err != nil {
		t.Fatalf("encryptString returned error: %v", err)
	}
	if _, err := base64.StdEncoding.DecodeString(encrypted); err != nil {
		t.Fatalf("encrypted value is not valid base64: %v", err)
	}

	decrypted, err := decryptString(encrypted, cfg)
	if err != nil {
		t.Fatalf("decryptString returned error: %v", err)
	}
	if decrypted != "1546c952" {
		t.Fatalf("decrypted = %q, want %q", decrypted, "1546c952")
	}
}

func TestDecryptAcceptsUnpaddedBase64(t *testing.T) {
	cfg := CryptoConfig{
		Seed: "python3806250511",
		IV:   "0625051106250511",
	}

	encrypted, err := encryptString("1546c952", cfg)
	if err != nil {
		t.Fatalf("encryptString returned error: %v", err)
	}
	encrypted = strings.TrimRight(encrypted, "=")

	decrypted, err := decryptString(encrypted, cfg)
	if err != nil {
		t.Fatalf("decryptString returned error: %v", err)
	}
	if decrypted != "1546c952" {
		t.Fatalf("decrypted = %q, want %q", decrypted, "1546c952")
	}
}

func TestDecryptObfuscatedResponseRemovesWireAndPlaintextPrefixes(t *testing.T) {
	cfg := DefaultConfig()
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load LA timezone: %v", err)
	}
	now := time.Date(2026, 5, 24, 0, 26, 0, 0, loc)
	wire := encryptObfuscatedFixture("2026-05-24 15:26:30", "python38x640026", cfg)

	decrypted, err := decryptResponseStringAt(wire, cfg, now)
	if err != nil {
		t.Fatalf("decryptResponseStringAt returned error: %v", err)
	}
	if decrypted != "2026-05-24 15:26:30" {
		t.Fatalf("decrypted = %q", decrypted)
	}
}

func encryptObfuscatedFixture(plain, seed string, cfg CryptoConfig) string {
	prefixBlock := []byte("0123456789abcdef")
	padded := pkcs7Pad(append(prefixBlock, []byte(plain)...), aesBlockSize)
	ciphertext, err := encryptCBC(padded, seed, cfg.IV)
	if err != nil {
		panic(err)
	}
	return "ABCDEF" + base64.StdEncoding.EncodeToString(ciphertext)
}
