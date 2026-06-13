package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	recordStatusPending       = "pending"
	recordStatusCreated       = "created"
	recordStatusCodeSubmitted = "code_submitted"
	recordStatusSucceeded     = "succeeded"
	recordStatusFailed        = "failed"
)

type State struct {
	InputFile string        `json:"inputFile"`
	CodeAPI   string        `json:"codeApi"`
	Records   []PhoneRecord `json:"records"`
}

type PhoneRecord struct {
	Phone        string    `json:"phone"`
	TaskID       uint      `json:"taskId,omitempty"`
	Status       string    `json:"status"`
	VerifyCode   string    `json:"verifyCode,omitempty"`
	LastError    string    `json:"lastError,omitempty"`
	TaskAttempts int       `json:"taskAttempts,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

var stateFileSaveMu sync.Mutex

func NewState(inputFile string, codeAPI string, phones []string) *State {
	now := time.Now()
	records := make([]PhoneRecord, 0, len(phones))
	for _, phone := range phones {
		records = append(records, PhoneRecord{
			Phone:     phone,
			Status:    recordStatusPending,
			UpdatedAt: now,
		})
	}
	return &State{
		InputFile: inputFile,
		CodeAPI:   codeAPI,
		Records:   records,
	}
}

func LoadOrCreateState(path string, inputFile string, data ImportData) (*State, error) {
	state, err := LoadStateFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return NewState(inputFile, data.CodeAPI, data.Phones), nil
	}
	state.InputFile = inputFile
	state.CodeAPI = data.CodeAPI
	state.MergePhones(data.Phones)
	return state, nil
}

func LoadStateFile(path string) (*State, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func SaveStateFile(path string, state *State) error {
	stateFileSaveMu.Lock()
	defer stateFileSaveMu.Unlock()

	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, raw, ".phonecodeworker-state-*.tmp")
}

func SaveFailedImportFile(path string, state *State) error {
	if strings.TrimSpace(path) == "" || state == nil {
		return nil
	}
	phones := state.failedPhones()
	if len(phones) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	var builder strings.Builder
	builder.WriteString(strings.TrimSpace(state.CodeAPI))
	builder.WriteByte('\n')
	for _, phone := range phones {
		builder.WriteString(phone)
		builder.WriteByte('\n')
	}
	return writeFileAtomic(path, []byte(builder.String()), ".phonecodeworker-failed-*.tmp")
}

func writeFileAtomic(path string, raw []byte, pattern string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpPath, path, err)
	}
	return nil
}

func (s *State) MergePhones(phones []string) {
	now := time.Now()
	seen := map[string]struct{}{}
	for _, rec := range s.Records {
		seen[rec.Phone] = struct{}{}
	}
	for _, phone := range phones {
		if _, ok := seen[phone]; ok {
			continue
		}
		s.Records = append(s.Records, PhoneRecord{
			Phone:     phone,
			Status:    recordStatusPending,
			UpdatedAt: now,
		})
		seen[phone] = struct{}{}
	}
}

func (s *State) activeRecord() *PhoneRecord {
	records := s.activeRecords()
	if len(records) == 0 {
		return nil
	}
	return records[0]
}

func (s *State) nextPendingRecord() *PhoneRecord {
	records := s.pendingRecords(1)
	if len(records) == 0 {
		return nil
	}
	return records[0]
}

func (s *State) activeRecords() []*PhoneRecord {
	records := []*PhoneRecord{}
	for i := range s.Records {
		rec := &s.Records[i]
		if rec.TaskID > 0 && !isTerminalRecordStatus(rec.Status) {
			records = append(records, rec)
		}
	}
	return records
}

func (s *State) pendingRecords(limit int) []*PhoneRecord {
	records := []*PhoneRecord{}
	for i := range s.Records {
		rec := &s.Records[i]
		if rec.Status == "" || rec.Status == recordStatusPending {
			records = append(records, rec)
			if limit > 0 && len(records) >= limit {
				break
			}
		}
	}
	return records
}

func (s *State) failedPhones() []string {
	phones := []string{}
	if s == nil {
		return phones
	}
	for _, rec := range s.Records {
		if rec.Status == recordStatusFailed {
			phone := strings.TrimSpace(rec.Phone)
			if phone != "" {
				phones = append(phones, phone)
			}
		}
	}
	return phones
}

func isTerminalRecordStatus(status string) bool {
	return status == recordStatusSucceeded || status == recordStatusFailed || status == recordStatusCodeSubmitted
}
