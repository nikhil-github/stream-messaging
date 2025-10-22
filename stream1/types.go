package stream1

import (
	"context"
	"time"
)

// ConsumerConfig holds configuration for a consumer
type ConsumerConfig struct {
	StreamName    string        // Stream name to consume from
	ConsumerGroup string        // Consumer group name for load sharing
	Subjects      []string      // Subjects to filter on
	FilterSubject string        // NATS filter subject pattern
	BatchSize     int           // Number of messages to pull in a batch
	BatchTimeout  time.Duration // Timeout for batch operations
	MaxDeliver    int           // Maximum delivery attempts
	MaxAckPending int           // Maximum unacknowledged messages
}

// Message represents a message from the stream
type Message interface {
	Subject() string            // Message subject
	Data() []byte               // Message payload
	Headers() map[string]string // Message headers
	MessageID() string          // Unique message ID for deduplication
	Timestamp() time.Time       // Message timestamp
	Ack() error                 // Acknowledge the message
	Nack() error                // Negative acknowledge the message
}

// PublishOptions provides options for publishing messages.
type PublishOptions struct {
	MessageID string            // MessageID is an optional unique identifier for the message
	Headers   map[string]string // Headers are optional headers to include with the message
}

// Publisher defines the interface for publishing messages
type Publisher interface {
	Publish(subject string, data []byte, opts PublishOptions) error
	Close() error
}

// Consumer defines the interface for consuming messages
type Consumer interface {
	PullBatch(ctx context.Context) (<-chan Message, error)
	Close() error
}

// Stream defines the main interface for the stream library
type Stream interface {
	NewPublisher(streamName string) (Publisher, error)
	NewConsumer(streamName string, config ConsumerConfig) (Consumer, error)
	Close() error
}
