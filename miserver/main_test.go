package main

import "testing"

func TestBuildListenAddresses(t *testing.T) {
	got := buildListenAddresses("127.0.0.2", 9999, 80, 8888)
	want := listenAddresses{
		Auth:   "127.0.0.2:9999",
		Upload: "127.0.0.2:80",
		Env:    "127.0.0.2:8888",
	}
	if got != want {
		t.Fatalf("addresses = %#v, want %#v", got, want)
	}
}
