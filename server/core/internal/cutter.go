package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Cutter 实现 io.Writer 接口
// 用于日志切割：路径为 director/[formats...]/level-{layout}.log（layout 为时间格式串时写入文件名，不再按日期建子目录）
type Cutter struct {
	level        string        // 日志级别(debug, info, warn, error, dpanic, panic, fatal)
	layout       string        // 时间格式 2006-01-02 15:04:05
	formats      []string      // 自定义参数([]string{Director,"2006-01-02", "business"(此参数可不写), level+".log"}
	director     string        // 日志文件夹
	retentionDay int           //日志保留天数
	file         *os.File      // 文件句柄
	mutex        *sync.RWMutex // 读写锁
}

type CutterOption func(*Cutter)

// CutterWithLayout 时间格式
func CutterWithLayout(layout string) CutterOption {
	return func(c *Cutter) {
		c.layout = layout
	}
}

// CutterWithFormats 格式化参数
func CutterWithFormats(format ...string) CutterOption {
	return func(c *Cutter) {
		if len(format) > 0 {
			c.formats = format
		}
	}
}

func NewCutter(director string, level string, retentionDay int, options ...CutterOption) *Cutter {
	rotate := &Cutter{
		level:        level,
		director:     director,
		retentionDay: retentionDay,
		mutex:        new(sync.RWMutex),
	}
	for i := 0; i < len(options); i++ {
		options[i](rotate)
	}
	return rotate
}

// Write satisfies the io.Writer interface. It writes to the
// appropriate file handle that is currently being used.
// If we have reached rotation time, the target file gets
// automatically rotated, and also purged if necessary.
func (c *Cutter) Write(bytes []byte) (n int, err error) {
	c.mutex.Lock()
	defer func() {
		if c.file != nil {
			_ = c.file.Close()
			c.file = nil
		}
		c.mutex.Unlock()
	}()
	parts := make([]string, 0, 2+len(c.formats))
	parts = append(parts, c.director)
	parts = append(parts, c.formats...)
	var filename string
	if c.layout != "" {
		filename = filepath.Join(append(parts, fmt.Sprintf("%s-%s.log", c.level, time.Now().Format(c.layout)))...)
	} else {
		filename = filepath.Join(append(parts, c.level+".log")...)
	}
	director := filepath.Dir(filename)
	err = os.MkdirAll(director, os.ModePerm)
	if err != nil {
		return 0, err
	}
	defer func() {
		err := removeNDaysFolders(c.director, c.retentionDay)
		if err != nil {
			fmt.Println("清理过期日志失败", err)
		}
	}()

	c.file, err = os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	return c.file.Write(bytes)
}

func (c *Cutter) Sync() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.file != nil {
		return c.file.Sync()
	}
	return nil
}

// 增加日志目录清理：删除过期子目录，并删除 director 下过期的 *.log（按文件修改时间）
func removeNDaysFolders(dir string, days int) error {
	if days <= 0 {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		if info.IsDir() {
			if info.ModTime().Before(cutoff) {
				return os.RemoveAll(path)
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".log") && info.ModTime().Before(cutoff) {
			return os.Remove(path)
		}
		return nil
	})
}
