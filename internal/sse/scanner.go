package sse

import (
	"bytes"
	"log"
	"strings"
)

const maxBufferSize = 4 * 1024 * 1024 // 4MB

var eventSep = []byte("\n\n")

type Scanner struct {
	buf bytes.Buffer
}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Feed(data []byte) []Event {
	// Normalize CRLF to LF
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	s.buf.Write(data)
	var events []Event
	for {
		idx := bytes.Index(s.buf.Bytes(), eventSep)
		if idx == -1 {
			break
		}
		raw := string(s.buf.Bytes()[:idx])
		s.buf.Next(idx + 2)
		events = append(events, parseEvent(raw))
	}
	// Prevent unbounded growth from malformed streams
	if s.buf.Len() > maxBufferSize {
		log.Printf("warning: SSE scanner buffer exceeded %d bytes, resetting", maxBufferSize)
		s.buf.Reset()
	}
	return events
}

func parseEvent(raw string) Event {
	var ev Event
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "event: ") {
			ev.Type = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			ev.Data = strings.TrimPrefix(line, "data: ")
		}
	}
	return ev
}
