package messaging

import "log/slog"

type Stream interface {
	NewPublisher(streamName string) (Publisher, error)
	NewConsumer(streamName string, cfg ConsumerConfig) (Consumer, error)
    // EnsureConsumer creates or updates a durable pull consumer with the given options.
    EnsureConsumer(streamName string, durable string, opts ConsumerOptions) error
	EnsureStream(streamName string, subjects []string) error
	Close() error
	WithLogger(logger *slog.Logger) Stream
}
