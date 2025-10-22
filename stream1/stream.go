package stream1

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// streamClient implements the Stream interface
type streamClient struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

// New creates a new Stream instance
func New(serverURL string) (Stream, error) {
	// Connect to NATS server
	nc, err := nats.Connect(serverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Get JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	return &streamClient{
		nc: nc,
		js: js,
	}, nil
}

// verifyStream checks that a stream exists in NATS
func (s *streamClient) verifyStream(streamName string) error {
	_, err := s.js.StreamInfo(streamName)
	if err != nil {
		return fmt.Errorf("stream %s does not exist: %w", streamName, err)
	}

	return nil
}

// NewPublisher creates a new publisher for the specified stream
func (s *streamClient) NewPublisher(streamName string) (Publisher, error) {
	// Verify stream exists
	if err := s.verifyStream(streamName); err != nil {
		return nil, err
	}

	return &publisher{js: s.js}, nil
}

// NewConsumer creates a new consumer with the given configuration
func (s *streamClient) NewConsumer(streamName string, config ConsumerConfig) (Consumer, error) {
	// Verify stream exists
	if err := s.verifyStream(streamName); err != nil {
		return nil, err
	}

	// Set default values
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.BatchTimeout <= 0 {
		config.BatchTimeout = 5 * time.Second
	}
	if config.MaxDeliver <= 0 {
		config.MaxDeliver = 5
	}
	if config.MaxAckPending <= 0 {
		config.MaxAckPending = 100
	}

	return &consumer{
		js:         s.js,
		streamName: streamName,
		config:     config,
	}, nil
}

// Close closes the stream connection
func (s *streamClient) Close() error {
	if s.nc != nil {
		s.nc.Close()
	}

	return nil
}
