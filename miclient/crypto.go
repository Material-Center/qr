package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const aesBlockSize = aes.BlockSize

type CryptoConfig struct {
	Seed               string
	IV                 string
	ResponseSeedPrefix string
	ResponseSkew       int
}

func DefaultConfig() CryptoConfig {
	return CryptoConfig{
		Seed:               "python3806250511",
		IV:                 "0625051106250511",
		ResponseSeedPrefix: "python38x64",
		ResponseSkew:       10,
	}
}

func deriveKey(seed string) []byte {
	sum := sha256.Sum256([]byte(seed))
	return sum[:]
}

func encryptString(plain string, cfg CryptoConfig) (string, error) {
	padded := pkcs7Pad([]byte(plain), aesBlockSize)
	encrypted, err := encryptCBC(padded, cfg.Seed, cfg.IV)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func decryptString(encoded string, cfg CryptoConfig) (string, error) {
	ciphertext, err := decodeBase64(encoded)
	if err != nil {
		return "", err
	}
	return decryptCBCToString(ciphertext, cfg.Seed, cfg.IV)
}

func decryptResponseString(encoded string, cfg CryptoConfig) (string, error) {
	return decryptResponseStringAt(encoded, cfg, time.Now())
}

func decryptResponseStringAt(encoded string, cfg CryptoConfig, now time.Time) (string, error) {
	if decrypted, err := decryptString(encoded, cfg); err == nil {
		return decrypted, nil
	}

	trimmed := strings.TrimSpace(encoded)
	if len(trimmed) <= 6 {
		return "", errors.New("obfuscated response is too short")
	}

	ciphertext, err := decodeBase64(trimmed[6:])
	if err != nil {
		return "", fmt.Errorf("decode obfuscated response: %w", err)
	}
	if len(ciphertext) == 0 || len(ciphertext)%aesBlockSize != 0 {
		return "", errors.New("ciphertext length must be a positive multiple of block size")
	}

	var lastErr error
	for _, seed := range responseSeeds(cfg, now) {
		decrypted, err := decryptCBC(ciphertext, seed, cfg.IV)
		if err != nil {
			lastErr = err
			continue
		}
		if len(decrypted) <= aesBlockSize {
			lastErr = errors.New("decrypted response is shorter than prefix block")
			continue
		}

		body := decrypted[aesBlockSize:]
		if !utf8.Valid(body) {
			lastErr = errors.New("decrypted response body is not valid utf-8")
			continue
		}
		return string(body), nil
	}

	if lastErr == nil {
		lastErr = errors.New("no response seeds were attempted")
	}
	return "", lastErr
}

func encryptCBC(padded []byte, seed string, iv string) ([]byte, error) {
	if len(iv) != aesBlockSize {
		return nil, fmt.Errorf("iv length = %d, want %d", len(iv), aesBlockSize)
	}

	block, err := aes.NewCipher(deriveKey(seed))
	if err != nil {
		return nil, err
	}

	encrypted := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, []byte(iv))
	mode.CryptBlocks(encrypted, padded)
	return encrypted, nil
}

func decryptCBCToString(ciphertext []byte, seed string, iv string) (string, error) {
	plain, err := decryptCBC(ciphertext, seed, iv)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func decryptCBC(ciphertext []byte, seed string, iv string) ([]byte, error) {
	if len(iv) != aesBlockSize {
		return nil, fmt.Errorf("iv length = %d, want %d", len(iv), aesBlockSize)
	}
	if len(ciphertext) == 0 || len(ciphertext)%aesBlockSize != 0 {
		return nil, errors.New("ciphertext length must be a positive multiple of block size")
	}

	block, err := aes.NewCipher(deriveKey(seed))
	if err != nil {
		return nil, err
	}

	plainPadded := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, []byte(iv))
	mode.CryptBlocks(plainPadded, ciphertext)

	plain, err := pkcs7Unpad(plainPadded, aesBlockSize)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

func responseSeeds(cfg CryptoConfig, now time.Time) []string {
	prefix := cfg.ResponseSeedPrefix
	if prefix == "" {
		prefix = "python38x64"
	}
	skew := cfg.ResponseSkew
	if skew <= 0 {
		skew = 10
	}

	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.FixedZone("America/Los_Angeles", -8*60*60)
	}
	center := now.In(loc)

	seeds := make([]string, 0, skew*2+1)
	for delta := -skew; delta <= skew; delta++ {
		t := center.Add(time.Duration(delta) * time.Minute)
		seeds = append(seeds, prefix+t.Format("1504"))
	}
	return seeds
}

func decodeBase64(encoded string) ([]byte, error) {
	normalized := strings.TrimSpace(encoded)
	if normalized == "" {
		return nil, errors.New("empty base64 value")
	}

	if out, err := base64.StdEncoding.DecodeString(normalized); err == nil {
		return out, nil
	}
	if out, err := base64.RawStdEncoding.DecodeString(normalized); err == nil {
		return out, nil
	}

	padded := normalized
	if remainder := len(padded) % 4; remainder != 0 {
		padded += strings.Repeat("=", 4-remainder)
	}
	return base64.StdEncoding.DecodeString(padded)
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	out := make([]byte, len(data)+padding)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(padding)
	}
	return out
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if blockSize <= 0 {
		return nil, errors.New("invalid block size")
	}
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, errors.New("invalid padded data length")
	}

	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize || padding > len(data) {
		return nil, errors.New("invalid padding")
	}
	for _, b := range data[len(data)-padding:] {
		if int(b) != padding {
			return nil, errors.New("invalid padding bytes")
		}
	}

	return data[:len(data)-padding], nil
}
