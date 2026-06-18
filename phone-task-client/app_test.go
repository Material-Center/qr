package main

import (
	"path/filepath"
	"testing"

	"phone-task-client/internal/domain"
	"phone-task-client/internal/store"
)

func TestEnsureDefaultAPITemplatesSeedsSendAndReceiveDefaults(t *testing.T) {
	st := newAppTestStore(t)

	if err := ensureDefaultAPITemplates(st); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	items, err := st.ListAPITemplates()
	if err != nil {
		t.Fatalf("list api templates: %v", err)
	}
	if !hasAPITemplate(items, domain.APITypePhoneSource, defaultPhoneSourceURL) {
		t.Fatalf("missing default phone source template: %#v", items)
	}
	if !hasAPITemplate(items, domain.APITypeCodeSource, defaultCodeSourceURL) {
		t.Fatalf("missing default code source template: %#v", items)
	}
}

func TestEnsureDefaultAPITemplatesIsIdempotent(t *testing.T) {
	st := newAppTestStore(t)

	if err := ensureDefaultAPITemplates(st); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}
	if err := ensureDefaultAPITemplates(st); err != nil {
		t.Fatalf("seed defaults again: %v", err)
	}

	items, err := st.ListAPITemplates()
	if err != nil {
		t.Fatalf("list api templates: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("template count = %d, want 2: %#v", len(items), items)
	}
}

func newAppTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "client.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}
