package source

import (
	"bufio"
	"fmt"
	"strings"
)

type PhoneEntry struct {
	Phone  string
	LineNo int
}

type TXTImport struct {
	CodeAPI string
	Phones  []PhoneEntry
}

func ParseTXTImport(raw string, firstLineCodeAPI bool) (TXTImport, error) {
	var out TXTImport
	seen := map[string]struct{}{}
	lineNo := 0
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		lineNo++
		line := cleanImportLine(scanner.Text())
		if line == "" {
			continue
		}
		if firstLineCodeAPI && out.CodeAPI == "" {
			out.CodeAPI = line
			continue
		}
		if !isElevenDigitPhone(line) {
			return TXTImport{}, fmt.Errorf("第%d行手机号格式不正确: %s", lineNo, line)
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out.Phones = append(out.Phones, PhoneEntry{Phone: line, LineNo: lineNo})
	}
	if err := scanner.Err(); err != nil {
		return TXTImport{}, err
	}
	if firstLineCodeAPI && out.CodeAPI == "" {
		return TXTImport{}, fmt.Errorf("导入文件第一行验证码API不能为空")
	}
	if len(out.Phones) == 0 {
		return TXTImport{}, fmt.Errorf("导入文件未包含手机号")
	}
	return out, nil
}

func cleanImportLine(raw string) string {
	line := strings.TrimSpace(raw)
	line = strings.TrimPrefix(line, "\ufeff")
	return strings.TrimSpace(line)
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
