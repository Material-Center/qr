package source

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var ErrPhoneNotReady = errors.New("手机号未就绪")

type VerifyCodeDetail struct {
	Source     string
	JSONParsed bool
}

func ExtractVerifyCodeDetail(body []byte) (string, VerifyCodeDetail) {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err == nil {
		for _, key := range []string{"data", "code", "verifyCode", "sms"} {
			if value, ok := obj[key]; ok {
				if code := firstDigitCode(fmt.Sprint(value)); code != "" {
					return code, VerifyCodeDetail{Source: "json." + key, JSONParsed: true}
				}
			}
		}
		return "", VerifyCodeDetail{Source: "json", JSONParsed: true}
	}
	if code := firstDigitCode(string(body)); code != "" {
		return code, VerifyCodeDetail{Source: "text", JSONParsed: false}
	}
	return "", VerifyCodeDetail{Source: "text", JSONParsed: false}
}

func ExtractPhoneFromAPIResponse(body []byte) (string, error) {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err == nil {
		for _, key := range []string{"data", "phone", "mobile"} {
			if value, ok := obj[key]; ok {
				if phone := firstElevenDigitPhone(fmt.Sprint(value)); phone != "" {
					return phone, nil
				}
			}
		}
		return "", ErrPhoneNotReady
	}
	if phone := firstElevenDigitPhone(string(body)); phone != "" {
		return phone, nil
	}
	return "", ErrPhoneNotReady
}

func firstDigitCode(raw string) string {
	re := regexp.MustCompile(`\d{4,8}`)
	return re.FindString(raw)
}

func firstElevenDigitPhone(raw string) string {
	fields := regexp.MustCompile(`\d{11}`).FindAllString(strings.TrimSpace(raw), -1)
	for _, field := range fields {
		if isElevenDigitPhone(field) {
			return field
		}
	}
	return ""
}
