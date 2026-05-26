package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type apiResponse struct {
	Code int              `json:"code"`
	Msg  string           `json:"msg"`
	Data importResultData `json:"data"`
}

type existsAPIResponse struct {
	Code int              `json:"code"`
	Msg  string           `json:"msg"`
	Data existsResultData `json:"data"`
}

type existsResultData struct {
	QQCacheRecordID uint   `json:"qqCacheRecordId"`
	QQNum           string `json:"qqNum"`
	Exists          bool   `json:"exists"`
}

type importResultData struct {
	QQCacheRecordID uint   `json:"qqCacheRecordId"`
	QQNum           string `json:"qqNum"`
	Action          string `json:"action"`
	Force           bool   `json:"force"`
}

type uploadItem struct {
	path  string
	qq    string
	pwd   string
	force bool
}

type accountListLine struct {
	QQNum  string
	Action string
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "qq-cache-import":
		runQQCacheImport(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func runQQCacheImport(args []string) {
	flags := flag.NewFlagSet("qq-cache-import", flag.ExitOnError)
	dir := flags.String("dir", ".", "zip文件目录")
	endpoint := flags.String("endpoint", "https://www.qq123qq.com/api/internalTool/qqCache/importZip", "内部工具导入接口地址")
	force := flags.Bool("force", false, "是否强制覆盖已存在账号的缓存字段")
	accountListOut := flags.String("account-list-out", "", "导入成功账号列表输出路径，默认输出到zip目录")
	timeout := flags.Duration("timeout", 2*time.Minute, "单个文件上传超时时间")
	if err := flags.Parse(args); err != nil {
		fatal(err)
	}

	files, err := filepath.Glob(filepath.Join(*dir, "*.zip"))
	if err != nil {
		fatal(err)
	}
	if len(files) == 0 {
		fatal(fmt.Errorf("目录下没有zip文件: %s", *dir))
	}

	client := &http.Client{Timeout: *timeout}
	var created, updated, skipped, failed int
	accountLines := make([]accountListLine, 0, len(files))
	for _, path := range files {
		qq, pwd, err := parseZipFileName(path)
		if err != nil {
			failed++
			fmt.Printf("[失败] %s: %v\n", filepath.Base(path), err)
			continue
		}
		if !*force {
			exists, err := checkExisting(client, *endpoint, qq)
			if err != nil {
				failed++
				fmt.Printf("[失败] %s: %v\n", filepath.Base(path), err)
				continue
			}
			if exists.Exists {
				skipped++
				accountLines = append(accountLines, accountListLine{QQNum: exists.QQNum, Action: "skipped"})
				fmt.Printf("[skipped] qq=%s recordId=%d file=%s\n", exists.QQNum, exists.QQCacheRecordID, filepath.Base(path))
				continue
			}
		}
		result, err := uploadZip(client, *endpoint, uploadItem{
			path:  path,
			qq:    qq,
			pwd:   pwd,
			force: *force,
		})
		if err != nil {
			failed++
			fmt.Printf("[失败] %s: %v\n", filepath.Base(path), err)
			continue
		}
		switch result.Action {
		case "created":
			created++
		case "updated":
			updated++
		case "skipped":
			skipped++
		}
		accountLines = append(accountLines, accountListLine{QQNum: result.QQNum, Action: result.Action})
		fmt.Printf("[%s] qq=%s recordId=%d file=%s\n", result.Action, result.QQNum, result.QQCacheRecordID, filepath.Base(path))
	}

	if len(accountLines) > 0 {
		outPath := buildAccountListOutPath(*dir, *accountListOut, func() string {
			return time.Now().Format("20060102_150405")
		})
		if err := writeAccountList(outPath, accountLines); err != nil {
			failed++
			fmt.Printf("[失败] 写入账号列表失败: %v\n", err)
		} else {
			fmt.Printf("账号列表: %s\n", outPath)
		}
	}
	fmt.Printf("完成: created=%d updated=%d skipped=%d failed=%d total=%d\n", created, updated, skipped, failed, len(files))
	if failed > 0 {
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "用法:")
	fmt.Fprintln(os.Stderr, "  internal-tool qq-cache-import -dir <zip目录> -endpoint <接口地址> [-force] [-account-list-out <txt路径>]")
}

func parseZipFileName(path string) (string, string, error) {
	name := filepath.Base(path)
	if !strings.EqualFold(filepath.Ext(name), ".zip") {
		return "", "", errors.New("不是zip文件")
	}
	base := strings.TrimSuffix(name, filepath.Ext(name))
	parts := strings.SplitN(base, "----", 2)
	if len(parts) != 2 {
		return "", "", errors.New("文件名格式应为 {qq uin}----{pwd}.zip")
	}
	qq := strings.TrimSpace(parts[0])
	pwd := strings.TrimSpace(parts[1])
	if qq == "" {
		return "", "", errors.New("文件名缺少QQ账号")
	}
	if pwd == "" {
		return "", "", errors.New("文件名缺少密码")
	}
	return qq, pwd, nil
}

func checkExisting(client *http.Client, endpoint string, qq string) (existsResultData, error) {
	checkURL := deriveExistsEndpoint(endpoint)
	parsed, err := url.Parse(checkURL)
	if err != nil {
		return existsResultData{}, err
	}
	query := parsed.Query()
	query.Set("qqNum", qq)
	parsed.RawQuery = query.Encode()

	resp, err := client.Get(parsed.String())
	if err != nil {
		return existsResultData{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return existsResultData{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return existsResultData{}, fmt.Errorf("检查已存在失败 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsedResp existsAPIResponse
	if err := json.Unmarshal(raw, &parsedResp); err != nil {
		return existsResultData{}, fmt.Errorf("解析检查响应失败: %w, body=%s", err, strings.TrimSpace(string(raw)))
	}
	if parsedResp.Code != 0 {
		return existsResultData{}, errors.New(parsedResp.Msg)
	}
	return parsedResp.Data, nil
}

func deriveExistsEndpoint(endpoint string) string {
	if strings.Contains(endpoint, "/importZip") {
		return strings.Replace(endpoint, "/importZip", "/exists", 1)
	}
	return strings.TrimRight(endpoint, "/") + "/exists"
}

func uploadZip(client *http.Client, endpoint string, item uploadItem) (importResultData, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("qqNum", item.qq); err != nil {
		return importResultData{}, err
	}
	if err := writer.WriteField("qqPwd", item.pwd); err != nil {
		return importResultData{}, err
	}
	if err := writer.WriteField("force", fmt.Sprintf("%t", item.force)); err != nil {
		return importResultData{}, err
	}
	file, err := os.Open(item.path)
	if err != nil {
		return importResultData{}, err
	}
	defer file.Close()
	part, err := writer.CreateFormFile("cacheZip", filepath.Base(item.path))
	if err != nil {
		return importResultData{}, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return importResultData{}, err
	}
	if err := writer.Close(); err != nil {
		return importResultData{}, err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, &body)
	if err != nil {
		return importResultData{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		return importResultData{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return importResultData{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return importResultData{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsed apiResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return importResultData{}, fmt.Errorf("解析响应失败: %w, body=%s", err, strings.TrimSpace(string(raw)))
	}
	if parsed.Code != 0 {
		return importResultData{}, errors.New(parsed.Msg)
	}
	return parsed.Data, nil
}

func buildAccountListOutPath(dir string, explicit string, timestamp func() string) string {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		return explicit
	}
	return filepath.Join(dir, fmt.Sprintf("qq_cache_import_accounts_%s.txt", timestamp()))
}

func writeAccountList(path string, lines []accountListLine) error {
	var b strings.Builder
	for _, line := range lines {
		qq := strings.TrimSpace(line.QQNum)
		if qq == "" {
			continue
		}
		action := strings.TrimSpace(line.Action)
		if action == "" {
			action = "-"
		}
		b.WriteString(qq)
		b.WriteString("----")
		b.WriteString(action)
		b.WriteString("\r\n")
	}
	if b.Len() == 0 {
		return nil
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
