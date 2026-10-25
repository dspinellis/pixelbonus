package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	mu  sync.Mutex
	f   *os.File
	now func() time.Time
}

func New(logDir string, now func() time.Time) (*Logger, error) {
	if now == nil {
		now = time.Now
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(logDir, "pixelbonus.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{f: f, now: now}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	return l.f.Close()
}

func (l *Logger) Log(level string, event string, fields map[string]string) {
	if l == nil || l.f == nil {
		return
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", k, fields[k]))
	}
	line := fmt.Sprintf("%s level=%s event=%s %s\n", l.now().UTC().Format(time.RFC3339), level, event, strings.Join(parts, " "))
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.f.WriteString(line)
}
