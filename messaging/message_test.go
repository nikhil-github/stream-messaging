package messaging

import "testing"

// These tests verify nil-safety of Message ack helpers without requiring NATS.

func TestMessageAckNil(t *testing.T) {
	m := &Message{}
	if err := m.Ack(); err == nil {
		t.Fatalf("expected error for nil message ack")
	}
}

func TestMessageNakNil(t *testing.T) {
	m := &Message{}
	if err := m.Nak(); err == nil {
		t.Fatalf("expected error for nil message nak")
	}
}

func TestMessageTermNil(t *testing.T) {
	m := &Message{}
	if err := m.Term(); err == nil {
		t.Fatalf("expected error for nil message term")
	}
}

func TestMessageInProgressNil(t *testing.T) {
	m := &Message{}
	if err := m.InProgress(); err == nil {
		t.Fatalf("expected error for nil message inprogress")
	}
}

func TestMessageAckSyncNil(t *testing.T) {
	m := &Message{}
	if err := m.AckSync(); err == nil {
		t.Fatalf("expected error for nil message acksync")
	}
}
