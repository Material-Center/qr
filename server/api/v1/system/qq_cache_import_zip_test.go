package system

import (
	"strings"
	"testing"
)

func TestResolveQQCacheImportZipCredentialsUsesFileName(t *testing.T) {
	qqNum, qqPwd, err := resolveQQCacheImportZipCredentials("", "", "10001----pwd123.zip")

	if err != nil {
		t.Fatalf("resolveQQCacheImportZipCredentials() error = %v", err)
	}
	if qqNum != "10001" {
		t.Fatalf("qqNum = %q, want %q", qqNum, "10001")
	}
	if qqPwd != "pwd123" {
		t.Fatalf("qqPwd = %q, want %q", qqPwd, "pwd123")
	}
}

func TestResolveQQCacheImportZipCredentialsPrefersFormPassword(t *testing.T) {
	qqNum, qqPwd, err := resolveQQCacheImportZipCredentials("10001", "formpwd", "10001----filepwd.zip")

	if err != nil {
		t.Fatalf("resolveQQCacheImportZipCredentials() error = %v", err)
	}
	if qqNum != "10001" {
		t.Fatalf("qqNum = %q, want %q", qqNum, "10001")
	}
	if qqPwd != "formpwd" {
		t.Fatalf("qqPwd = %q, want %q", qqPwd, "formpwd")
	}
}

func TestResolveQQCacheImportZipCredentialsRejectsMismatchedQQ(t *testing.T) {
	_, _, err := resolveQQCacheImportZipCredentials("10001", "", "10002----pwd123.zip")

	if err == nil {
		t.Fatal("expected mismatched qq error")
	}
	if !strings.Contains(err.Error(), "文件名账号与请求账号不一致") {
		t.Fatalf("error = %q, want mismatch message", err.Error())
	}
}

func TestResolveQQCacheImportZipCredentialsKeepsLegacyNameWithoutPassword(t *testing.T) {
	qqNum, qqPwd, err := resolveQQCacheImportZipCredentials("", "", "cache.zip")

	if err != nil {
		t.Fatalf("resolveQQCacheImportZipCredentials() error = %v", err)
	}
	if qqNum != "" {
		t.Fatalf("qqNum = %q, want empty", qqNum)
	}
	if qqPwd != "" {
		t.Fatalf("qqPwd = %q, want empty", qqPwd)
	}
}
