package messaging

import (
    "context"
    "testing"
)

// Validate subject required behavior without needing a live NATS server.
func TestPublisherSubjectRequired(t *testing.T) {
    p := &jsPublisher{}
    if err := p.Publish(context.Background(), []byte("hi"), nil); err == nil {
        t.Fatalf("expected error when opts is nil")
    }
    if err := p.Publish(context.Background(), []byte("hi"), &PublishOptions{}); err == nil {
        t.Fatalf("expected error when subject is empty")
    }
}
