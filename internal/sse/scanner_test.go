package sse

import "testing"

func TestScanSingleEvent(t *testing.T) {
	s := NewScanner()
	input := []byte("event: message_start\ndata: {\"type\":\"message_start\"}\n\n")
	events := s.Feed(input)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type != "message_start" {
		t.Errorf("type = %q, want %q", events[0].Type, "message_start")
	}
}

func TestScanMultipleEvents(t *testing.T) {
	s := NewScanner()
	input := []byte("event: content_block_start\ndata: {}\n\nevent: message_delta\ndata: {\"usage\":{}}\n\n")
	events := s.Feed(input)
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[1].Type != "message_delta" {
		t.Errorf("type = %q, want %q", events[1].Type, "message_delta")
	}
}

func TestScanPartialEvent(t *testing.T) {
	s := NewScanner()
	events1 := s.Feed([]byte("event: message_start\nda"))
	if len(events1) != 0 {
		t.Fatalf("got %d events from partial, want 0", len(events1))
	}
	events2 := s.Feed([]byte("ta: {\"type\":\"message_start\"}\n\n"))
	if len(events2) != 1 {
		t.Fatalf("got %d events after completion, want 1", len(events2))
	}
}

func TestScanNoEventType(t *testing.T) {
	s := NewScanner()
	input := []byte("data: {\"choices\":[{\"delta\":{}}]}\n\n")
	events := s.Feed(input)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type != "" {
		t.Errorf("type = %q, want empty", events[0].Type)
	}
}

func TestScanDoneSignal(t *testing.T) {
	s := NewScanner()
	input := []byte("data: [DONE]\n\n")
	events := s.Feed(input)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Data != "[DONE]" {
		t.Errorf("data = %q, want %q", events[0].Data, "[DONE]")
	}
}
