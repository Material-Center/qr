package system

import (
	"errors"
	"strings"
)

const (
	AccountTypeDefault = "default"
	AccountTypePC      = "pc"
)

type QQCacheAccountTypeOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func SupportedQQCacheAccountTypes() []QQCacheAccountTypeOption {
	return []QQCacheAccountTypeOption{
		{Value: AccountTypeDefault, Label: "默认账号"},
		{Value: AccountTypePC, Label: "PC号"},
	}
}

func (s *QQCacheService) AccountTypes() []QQCacheAccountTypeOption {
	return SupportedQQCacheAccountTypes()
}

func NormalizeQQCacheAccountType(value string) string {
	value = strings.TrimSpace(value)
	if isSupportedQQCacheAccountType(value) {
		return value
	}
	return AccountTypeDefault
}

func ValidateQQCacheAccountType(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || isSupportedQQCacheAccountType(value) {
		return nil
	}
	return errors.New("账号类型不支持")
}

func SanitizeQQCacheAccountTypes(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !isSupportedQQCacheAccountType(value) {
			return nil, errors.New("账号类型不支持")
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func isSupportedQQCacheAccountType(value string) bool {
	switch value {
	case AccountTypeDefault, AccountTypePC:
		return true
	default:
		return false
	}
}
