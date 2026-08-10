package system

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQQCacheAccountTypeNormalize(t *testing.T) {
	require.Equal(t, AccountTypeDefault, NormalizeQQCacheAccountType(""))
	require.Equal(t, AccountTypeDefault, NormalizeQQCacheAccountType("  "))
	require.Equal(t, AccountTypeDefault, NormalizeQQCacheAccountType("unknown"))
	require.Equal(t, AccountTypePC, NormalizeQQCacheAccountType(" pc "))
}

func TestQQCacheAccountTypeValidate(t *testing.T) {
	require.NoError(t, ValidateQQCacheAccountType(""))
	require.NoError(t, ValidateQQCacheAccountType(AccountTypeDefault))
	require.NoError(t, ValidateQQCacheAccountType(AccountTypePC))
	require.Error(t, ValidateQQCacheAccountType("mobile"))
}

func TestSanitizeQQCacheAccountTypes(t *testing.T) {
	types, err := SanitizeQQCacheAccountTypes([]string{" pc ", AccountTypeDefault, AccountTypePC})
	require.NoError(t, err)
	require.Equal(t, []string{AccountTypePC, AccountTypeDefault}, types)

	_, err = SanitizeQQCacheAccountTypes([]string{"mobile"})
	require.Error(t, err)
}
