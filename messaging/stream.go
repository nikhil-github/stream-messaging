package messaging

import "log/slog"

type Stream interface {
	NewPublisher(streamName string) (Publisher, error)
	NewConsumer(streamName string, cfg ConsumerConfig) (Consumer, error)
	EnsureStream(streamName string, subjects []string) error
	Close() error
	WithLogger(logger *slog.Logger) Stream
}
