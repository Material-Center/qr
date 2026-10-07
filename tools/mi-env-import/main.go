package main

// Re-runnable source-to-server migration helper.
import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	srcBase = "http://39.108.96.33:8888"
	dstURL  = "http://210.16.170.132:1111/api/internalTool/miEnv/import_env"
	iv      = "0625051106250511"
	key     = "cd5d1c1b4bd95fcb561d2a3f2b5407de82b9088d8b1f4eb3ebce9c28331ef42f"
	page    = 500
)

func main() {
	if len(os.Args) != 2 || strings.TrimSpace(os.Args[1]) == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./miserver/import_env_tool.go <deviceid>")
		os.Exit(2)
	}
	device := strings.TrimSpace(os.Args[1])
	h := &http.Client{Timeout: 60 * time.Second}
	total, inserted, updated := 0, 0, 0
	for offset := 0; ; offset += page {
		items, err := sourcePage(h, device, offset)
		if err != nil {
			panic(fmt.Errorf("device %s offset %d: %w", device, offset, err))
		}
		if len(items) == 0 {
			break
		}
		for _, item := range items {
			if str(item["设备ID"]) != device {
				panic(fmt.Errorf("device %s returned mismatched record", device))
			}
			if typ := str(item["类型"]); typ == "" {
				panic(fmt.Errorf("device %s returned empty type", device))
			}
		}
		result, err := targetImportBatch(h, items)
		if err != nil {
			panic(fmt.Errorf("device %s offset %d import: %w", device, offset, err))
		}
		inserted += asInt(result["inserted"])
		updated += asInt(result["updated"])
		total += len(items)
		fmt.Printf("device=%s offset=%d processed=%d inserted=%d updated=%d\n", device, offset, len(items), asInt(result["inserted"]), asInt(result["updated"]))
		if len(items) < page {
			break
		}
	}
	fmt.Printf("completed source=%d inserted=%d updated=%d\n", total, inserted, updated)
}

func sourcePage(h *http.Client, device string, offset int) ([]map[string]any, error) {
	now := time.Now()
	payload := map[string]any{"设备ID": device, "limit": page, "offset": offset, "排序": "id DESC", "key": encrypt([]byte("06250511"), now)}
	data, err := request(h, srcBase+"/query_env", payload, "")
	if err != nil {
		return nil, err
	}
	var result struct {
		Code int              `json:"code"`
		Data []map[string]any `json:"data"`
		Msg  string           `json:"msg"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("business code=%d msg=%s", result.Code, result.Msg)
	}
	return result.Data, nil
}

func targetImportBatch(h *http.Client, items []map[string]any) (map[string]any, error) {
	data, err := request(h, dstURL+"_batch", map[string]any{"records": items}, key)
	if err != nil {
		return nil, err
	}
	var result struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
		Msg  string         `json:"msg"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("business code=%d msg=%s", result.Code, result.Msg)
	}
	return result.Data, nil
}

func request(h *http.Client, url string, payload map[string]any, headerKey string) ([]byte, error) {
	now := time.Now()
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]string{"data": encrypt(raw, now)})
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if headerKey != "" {
		req.Header.Set("X-MI-Internal-Key", headerKey)
	}
	resp, err := h.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Code int    `json:"code"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return nil, err
	}
	if envelope.Code != 0 && envelope.Code != http.StatusOK {
		return nil, fmt.Errorf("outer code=%d body=%s", envelope.Code, strings.TrimSpace(string(responseBody)))
	}
	plain, err := decrypt(envelope.Data, time.Now())
	if err != nil {
		return nil, err
	}
	return []byte(plain), nil
}

func encrypt(plain []byte, now time.Time) string {
	prefix := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, prefix); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(crypt(pad(append(prefix, plain...)), seed(now), true))
}

func decrypt(encoded string, now time.Time) (string, error) {
	values := []string{strings.TrimSpace(encoded)}
	if len(values[0]) > 6 {
		values = append(values, values[0][6:])
	}
	center := now.In(loc())
	var last error
	for _, value := range values {
		ciphertext, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			last = err
			continue
		}
		for _, delta := range []int{0, -1, 1, -2, 2} {
			plain, err := unpad(crypt(ciphertext, seed(center.Add(time.Duration(delta)*time.Minute)), false))
			if err != nil || len(plain) <= aes.BlockSize {
				last = err
				continue
			}
			return string(plain[aes.BlockSize:]), nil
		}
	}
	if last == nil {
		last = errors.New("unable to decrypt response")
	}
	return "", last
}

func seed(t time.Time) string { return "python38x64" + t.In(loc()).Format("1504") }
func loc() *time.Location {
	l, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}
	return l
}
func crypt(in []byte, s string, enc bool) []byte {
	sum := sha256.Sum256([]byte(s))
	b, err := aes.NewCipher(sum[:])
	if err != nil {
		panic(err)
	}
	out := make([]byte, len(in))
	if enc {
		cipher.NewCBCEncrypter(b, []byte(iv)).CryptBlocks(out, in)
	} else {
		cipher.NewCBCDecrypter(b, []byte(iv)).CryptBlocks(out, in)
	}
	return out
}
func pad(in []byte) []byte {
	n := aes.BlockSize - len(in)%aes.BlockSize
	return append(in, bytes.Repeat([]byte{byte(n)}, n)...)
}
func unpad(in []byte) ([]byte, error) {
	if len(in) == 0 || len(in)%aes.BlockSize != 0 {
		return nil, errors.New("invalid padding length")
	}
	n := int(in[len(in)-1])
	if n == 0 || n > aes.BlockSize || n > len(in) {
		return nil, errors.New("invalid padding")
	}
	for _, b := range in[len(in)-n:] {
		if int(b) != n {
			return nil, errors.New("invalid padding bytes")
		}
	}
	return in[:len(in)-n], nil
}
func str(v any) string { s, _ := v.(string); return s }

func asInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}
