package system

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

const miEnvBlockSize = aes.BlockSize

type miEnvCryptoConfig struct {
	Seed               string
	IV                 string
	ResponseSeedPrefix string
	ResponseSkew       int
}

func defaultMIEnvCrypto() miEnvCryptoConfig {
	return miEnvCryptoConfig{Seed: "06250511", IV: "0625051106250511", ResponseSeedPrefix: "python38x64", ResponseSkew: 2}
}

func decryptMIEnvRequest(encoded string, cfg miEnvCryptoConfig, now time.Time) (string, error) {
	ciphertext, err := decodeMIEnvBase64(encoded)
	if err != nil {
		return "", err
	}
	var lastErr error
	for _, seed := range miEnvResponseSeeds(cfg, now) {
		plain, err := decryptMIEnvCBC(ciphertext, seed, cfg.IV)
		if err != nil {
			lastErr = err
			continue
		}
		if len(plain) <= miEnvBlockSize || !utf8.Valid(plain[miEnvBlockSize:]) {
			lastErr = errors.New("invalid encrypted MI environment request")
			continue
		}
		return string(plain[miEnvBlockSize:]), nil
	}
	if lastErr == nil {
		lastErr = errors.New("no MI environment response seeds matched")
	}
	return "", lastErr
}

func encryptMIEnvResponse(plain string, cfg miEnvCryptoConfig, now time.Time) (string, error) {
	prefix := make([]byte, miEnvBlockSize)
	if _, err := io.ReadFull(rand.Reader, prefix); err != nil {
		return "", err
	}
	padded := miEnvPKCS7Pad(append(prefix, []byte(plain)...), miEnvBlockSize)
	ciphertext, err := encryptMIEnvCBC(padded, miEnvResponseSeed(cfg, now), cfg.IV)
	if err != nil {
		return "", err
	}
	wirePrefix := make([]byte, 6)
	if _, err := io.ReadFull(rand.Reader, wirePrefix); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(wirePrefix)[:6] + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func miEnvResponseSeed(cfg miEnvCryptoConfig, now time.Time) string {
	return cfg.ResponseSeedPrefix + now.In(miEnvLosAngeles()).Format("1504")
}

func miEnvResponseSeeds(cfg miEnvCryptoConfig, now time.Time) []string {
	skew := cfg.ResponseSkew
	if skew <= 0 {
		skew = 2
	}
	center := now.In(miEnvLosAngeles())
	seeds := make([]string, 0, skew*2+1)
	seeds = append(seeds, cfg.ResponseSeedPrefix+center.Format("1504"))
	for i := 1; i <= skew; i++ {
		seeds = append(seeds, cfg.ResponseSeedPrefix+center.Add(-time.Duration(i)*time.Minute).Format("1504"))
		seeds = append(seeds, cfg.ResponseSeedPrefix+center.Add(time.Duration(i)*time.Minute).Format("1504"))
	}
	return seeds
}

func encryptMIEnvCBC(padded []byte, seed, iv string) ([]byte, error) {
	if len(iv) != miEnvBlockSize {
		return nil, fmt.Errorf("invalid IV length: %d", len(iv))
	}
	block, err := aes.NewCipher(miEnvKey(seed))
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, []byte(iv)).CryptBlocks(out, padded)
	return out, nil
}

func decryptMIEnvCBC(ciphertext []byte, seed, iv string) ([]byte, error) {
	if len(ciphertext) == 0 || len(ciphertext)%miEnvBlockSize != 0 {
		return nil, errors.New("invalid ciphertext length")
	}
	if len(iv) != miEnvBlockSize {
		return nil, fmt.Errorf("invalid IV length: %d", len(iv))
	}
	block, err := aes.NewCipher(miEnvKey(seed))
	if err != nil {
		return nil, err
	}
	plain := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, []byte(iv)).CryptBlocks(plain, ciphertext)
	return miEnvPKCS7Unpad(plain, miEnvBlockSize)
}

func miEnvKey(seed string) []byte { sum := sha256.Sum256([]byte(seed)); return sum[:] }

func miEnvPKCS7Pad(data []byte, size int) []byte {
	n := size - len(data)%size
	out := make([]byte, len(data)+n)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(n)
	}
	return out
}

func miEnvPKCS7Unpad(data []byte, size int) ([]byte, error) {
	if len(data) == 0 || len(data)%size != 0 {
		return nil, errors.New("invalid PKCS7 data")
	}
	n := int(data[len(data)-1])
	if n == 0 || n > size || n > len(data) {
		return nil, errors.New("invalid PKCS7 padding")
	}
	for _, b := range data[len(data)-n:] {
		if int(b) != n {
			return nil, errors.New("invalid PKCS7 padding")
		}
	}
	return data[:len(data)-n], nil
}

func decodeMIEnvBase64(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("empty base64 value")
	}
	if out, err := base64.StdEncoding.DecodeString(value); err == nil {
		return out, nil
	}
	if out, err := base64.RawStdEncoding.DecodeString(value); err == nil {
		return out, nil
	}
	if n := len(value) % 4; n != 0 {
		value += strings.Repeat("=", 4-n)
	}
	return base64.StdEncoding.DecodeString(value)
}

func miEnvLosAngeles() *time.Location {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return time.FixedZone("America/Los_Angeles", -8*60*60)
	}
	return loc
}
