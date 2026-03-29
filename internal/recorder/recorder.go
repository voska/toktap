package recorder

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Record struct {
	ID              string            `json:"id"`
	Timestamp       time.Time         `json:"timestamp"`
	Provider        string            `json:"provider"`
	Model           string            `json:"model"`
	Device          string            `json:"device"`
	Harness         string            `json:"harness"`
	RequestPath     string            `json:"request_path"`
	RequestMethod   string            `json:"request_method"`
	RequestHeaders  map[string]string `json:"request_headers"`
	RequestBody     json.RawMessage   `json:"request_body"`
	ResponseStatus  int               `json:"response_status"`
	ResponseHeaders map[string]string `json:"response_headers"`
	ResponseBody    json.RawMessage   `json:"response_body"`
	IsStreaming     bool              `json:"is_streaming"`
	SSEEvents       []SSEEvent        `json:"sse_events,omitempty"`
	InputTokens     int64             `json:"input_tokens"`
	OutputTokens    int64             `json:"output_tokens"`
	DurationMs      int64             `json:"duration_ms"`
}

type SSEEvent struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type Recorder struct {
	dir string
	mu  sync.Mutex
	f   *os.File
	day string
}

func New(dir string) (*Recorder, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating recorder dir: %w", err)
	}
	return &Recorder{dir: dir}, nil
}

func (r *Recorder) Write(rec Record) {
	data, err := json.Marshal(rec)
	if err != nil {
		log.Printf("recorder: marshal error: %v", err)
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	day := rec.Timestamp.Format("2006-01-02")
	if err := r.ensureFile(day); err != nil {
		log.Printf("recorder: file error: %v", err)
		return
	}

	data = append(data, '\n')
	if _, err := r.f.Write(data); err != nil {
		log.Printf("recorder: write error: %v", err)
	}
}

func (r *Recorder) ensureFile(day string) error {
	if r.f != nil && r.day == day {
		return nil
	}
	if r.f != nil {
		_ = r.f.Close()
	}
	path := filepath.Join(r.dir, day+".jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	r.f = f
	r.day = day
	return nil
}

func (r *Recorder) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f != nil {
		_ = r.f.Close()
	}
}
