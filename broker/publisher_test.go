package broker

import (
	"context"
	"errors"
	"testing"
)

// Validate subject required behavior without needing a live NATS server.
func TestPublisherSubjectRequired(t *testing.T) {
	p := &jsPublisher{}
	
	// Empty subject should return error
	err := p.Publish(context.Background(), "", []byte("hi"), nil)
	if err == nil {
		t.Fatalf("expected error when subject is empty")
	}
	if !errors.Is(err, ErrSubjectRequired) {
		t.Fatalf("expected ErrSubjectRequired, got: %v", err)
	}
}
