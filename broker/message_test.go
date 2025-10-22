package broker

import (
	"errors"
	"testing"
)

// These tests verify nil-safety of Message ack helpers without requiring NATS.

func TestMessageAckNil(t *testing.T) {
	m := &Message{}
	err := m.Ack()
	if err == nil {
		t.Fatalf("expected error for nil message ack")
	}
	if !errors.Is(err, ErrNilMessage) {
		t.Fatalf("expected ErrNilMessage, got: %v", err)
	}
}

func TestMessageNakNil(t *testing.T) {
	m := &Message{}
	err := m.Nak()
	if err == nil {
		t.Fatalf("expected error for nil message nak")
	}
	if !errors.Is(err, ErrNilMessage) {
		t.Fatalf("expected ErrNilMessage, got: %v", err)
	}
}

func TestMessageTermNil(t *testing.T) {
	m := &Message{}
	err := m.Term()
	if err == nil {
		t.Fatalf("expected error for nil message term")
	}
	if !errors.Is(err, ErrNilMessage) {
		t.Fatalf("expected ErrNilMessage, got: %v", err)
	}
}

func TestMessageInProgressNil(t *testing.T) {
	m := &Message{}
	err := m.InProgress()
	if err == nil {
		t.Fatalf("expected error for nil message inprogress")
	}
	if !errors.Is(err, ErrNilMessage) {
		t.Fatalf("expected ErrNilMessage, got: %v", err)
	}
}

func TestMessageAckSyncNil(t *testing.T) {
	m := &Message{}
	err := m.AckSync()
	if err == nil {
		t.Fatalf("expected error for nil message acksync")
	}
	if !errors.Is(err, ErrNilMessage) {
		t.Fatalf("expected ErrNilMessage, got: %v", err)
	}
}
