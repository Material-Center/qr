package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type ImportData struct {
	CodeAPI string
	Phones  []string
}

func cleanImportLine(raw string) string {
	line := strings.TrimSpace(raw)
	line = strings.TrimPrefix(line, "\ufeff")
	return strings.TrimSpace(line)
}

func cleanCodeAPI(raw string) string {
	return cleanImportLine(raw)
}

func LoadImportFile(path string) (ImportData, error) {
	f, err := os.Open(path)
	if err != nil {
		return ImportData{}, err
	}
	defer f.Close()

	var out ImportData
	lineNo := 0
	seen := map[string]struct{}{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNo++
		line := cleanImportLine(scanner.Text())
		if line == "" {
			continue
		}
		if out.CodeAPI == "" {
			out.CodeAPI = line
			continue
		}
		if !isElevenDigitPhone(line) {
			return ImportData{}, fmt.Errorf("第%d行手机号格式不正确: %s", lineNo, line)
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out.Phones = append(out.Phones, line)
	}
	if err := scanner.Err(); err != nil {
		return ImportData{}, err
	}
	if out.CodeAPI == "" {
		return ImportData{}, fmt.Errorf("导入文件第一行验证码API不能为空")
	}
	if len(out.Phones) == 0 {
		return ImportData{}, fmt.Errorf("导入文件未包含手机号")
	}
	return out, nil
}

func isElevenDigitPhone(phone string) bool {
	if len(phone) != 11 {
		return false
	}
	for _, ch := range phone {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
