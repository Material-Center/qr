package main

import (
	"encoding/json"
	"errors"
	"os"
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
	Phone      string    `json:"phone"`
	TaskID     uint      `json:"taskId,omitempty"`
	Status     string    `json:"status"`
	VerifyCode string    `json:"verifyCode,omitempty"`
	LastError  string    `json:"lastError,omitempty"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

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
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
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
	for i := range s.Records {
		rec := &s.Records[i]
		if rec.TaskID > 0 && !isTerminalRecordStatus(rec.Status) {
			return rec
		}
	}
	return nil
}

func (s *State) nextPendingRecord() *PhoneRecord {
	for i := range s.Records {
		rec := &s.Records[i]
		if rec.Status == "" || rec.Status == recordStatusPending {
			return rec
		}
	}
	return nil
}

func isTerminalRecordStatus(status string) bool {
	return status == recordStatusSucceeded || status == recordStatusFailed
}
