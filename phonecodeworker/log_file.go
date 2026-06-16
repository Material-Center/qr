package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type dailyLogWriter struct {
	mu          sync.Mutex
	dir         string
	prefix      string
	now         func() time.Time
	currentDate string
	currentPath string
	file        *os.File
}

func newDailyLogWriter(dir string, prefix string, now func() time.Time) (*dailyLogWriter, error) {
	if now == nil {
		now = time.Now
	}
	w := &dailyLogWriter{
		dir:    dir,
		prefix: prefix,
		now:    now,
	}
	if err := w.rotateLocked(now()); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *dailyLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := w.now()
	date := now.Format("20060102")
	if w.file == nil || date != w.currentDate {
		if err := w.rotateLocked(now); err != nil {
			return 0, err
		}
	}
	return w.file.Write(p)
}

func (w *dailyLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *dailyLogWriter) Path() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.currentPath
}

func (w *dailyLogWriter) rotateLocked(now time.Time) error {
	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return err
	}
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}
	path, file, err := createUniqueLogFile(w.dir, w.prefix, now)
	if err != nil {
		return err
	}
	w.currentDate = now.Format("20060102")
	w.currentPath = path
	w.file = file
	return nil
}

func createUniqueLogFile(dir string, prefix string, now time.Time) (string, *os.File, error) {
	base := fmt.Sprintf("%s-%s", prefix, now.Format("20060102-150405"))
	for i := 0; i < 1000; i++ {
		name := base + ".log"
		if i > 0 {
			name = fmt.Sprintf("%s-%d.log", base, i+1)
		}
		path := filepath.Join(dir, name)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			return path, file, nil
		}
		if !os.IsExist(err) {
			return "", nil, err
		}
	}
	return "", nil, fmt.Errorf("create unique log file in %s: too many collisions", dir)
}

func defaultLogDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "logs"
	}
	return filepath.Join(filepath.Dir(exe), "logs")
}
