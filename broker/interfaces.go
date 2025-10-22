package broker

import (
	"context"
	"time"
)

// Client is the main interface for interacting with a messaging system.
// It provides methods to create publishers and consumers for specific streams.
type Client interface {
	// NewPublisher creates a new publisher for the specified stream
	NewPublisher(streamName string) (Publisher, error)

	// NewConsumer creates a new consumer for the specified stream
	NewConsumer(streamName string, cfg ConsumerConfig) (Consumer, error)

	// EnsureStream ensures that a stream exists with the given subjects.
	// If the stream already exists, this is a no-op.
	// Note: Some implementations may not support this if streams are managed externally.
	EnsureStream(streamName string, subjects []string) error

	// Close closes the client and releases any resources
	Close() error
}

// Publisher defines the interface for publishing messages to a stream
type Publisher interface {
	// Publish sends a message to the specified subject with optional metadata
	Publish(ctx context.Context, subject string, data []byte, opts *PublishOptions) error

	// Close closes the publisher and releases any resources
	Close() error
}

// Consumer defines the interface for consuming messages from a stream
type Consumer interface {
	// PullBatch pulls a batch of messages from the stream.
	// Returns a slice of messages or an error if the pull fails.
	//
	// Timeout behavior:
	//   - If no messages are available within the timeout, returns empty slice (not an error)
	//   - If context is cancelled/expired, returns context.Err()
	//   - Only returns error for actual failures (connection issues, etc.)
	//
	// Example:
	//   messages, err := consumer.PullBatch(ctx)
	//   if err != nil {
	//       // Real error (connection issue, etc.)
	//       log.Fatal(err)
	//   }
	//   if len(messages) == 0 {
	//       // No messages available, try again later
	//   }
	PullBatch(ctx context.Context) ([]*Message, error)

	// Subscribe starts consuming messages continuously via a channel.
	// The channel will be closed when the context is cancelled or an error occurs.
	// This is optional and may not be implemented by all providers.
	Subscribe(ctx context.Context) (<-chan *Message, error)

	// Close closes the consumer and releases any resources
	Close() error
}

// PublishOptions provides optional parameters for publishing messages
type PublishOptions struct {
	// MessageID is an optional unique identifier for deduplication
	MessageID string

	// Headers are optional key-value pairs to include with the message
	Headers map[string]string
}

// ConsumerConfig holds configuration for creating a consumer
type ConsumerConfig struct {
	// Subject is the subject/topic to consume from (required)
	Subject string

	// BatchSize is the number of messages to pull in each batch
	BatchSize int

	// BatchTimeout is the timeout for individual batch fetch operations
	BatchTimeout time.Duration

	// AckWait is the duration before a message is redelivered if not acknowledged
	AckWait time.Duration

	// Durable is the name of the durable consumer for persistence across restarts
	Durable string

	// MaxDeliver is the maximum number of delivery attempts (0 = unlimited)
	MaxDeliver int

	// MaxAckPending is the maximum number of unacknowledged messages allowed
	MaxAckPending int

	// FilterSubject is an optional subject filter pattern
	FilterSubject string
}
