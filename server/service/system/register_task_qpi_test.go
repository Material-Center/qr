package system

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestFetchKuaidailiSocks5(t *testing.T) {
	secretID := "oukz5ljaxxemzy2dhagx"
	secretKey := "0tikidzbvicyyhi2evxu2ulrsvz29mw0"
	client := &http.Client{Timeout: 10 * time.Second}
	got, err := fetchKuaidailiSocks5(client, secretID, secretKey, "211300", "联通")
	if err != nil {
		t.Fatalf("real api request failed: %v", err)
	}
	if !strings.HasPrefix(got, "socks5://") {
		t.Fatalf("unexpected proxy scheme: %s", got)
	}
	parsed, parseErr := url.Parse(got)
	if parseErr != nil {
		t.Fatalf("parse proxy url failed: %v", parseErr)
	}
	if _, _, splitErr := net.SplitHostPort(parsed.Host); splitErr != nil {
		t.Fatalf("unexpected proxy host:port: %s, err=%v", got, splitErr)
	}
	t.Logf("got: %s", got)
}

func TestFetchPingzanSocks5WithAuth(t *testing.T) {
	const no = "20260406122909519389"
	const secret = "o77kak904ibon"

	client := &http.Client{Timeout: 3 * time.Second}
	got, err := fetchPingzanSocks5(client, no, secret, "")
	if err != nil {
		t.Fatalf("fetch pingzan failed: %v", err)
	}
	if !strings.HasPrefix(got, "socks5://") {
		t.Fatalf("unexpected proxy scheme: %s", got)
	}
	t.Logf("got: %s", got)
}
