package broker

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type jetStreamClient struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	closed bool
}

// newJetStreamClient creates a new NATS JetStream client.
// This is an internal function. Use broker.NewClient() instead.
func newJetStreamClient(url string) (Client, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("initialize JetStream: %w", err)
	}

	return &jetStreamClient{nc: nc, js: js}, nil
}

func (c *jetStreamClient) EnsureStream(streamName string, subjects []string) error {
	if c.closed {
		return ErrConnectionClosed
	}

	_, err := c.js.StreamInfo(streamName)
	if err == nil {
		// Stream already exists
		return nil
	}

	if err == nats.ErrStreamNotFound {
		_, err = c.js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: subjects,
			Storage:  nats.FileStorage,
			Replicas: 1,
		})
		return err
	}

	return fmt.Errorf("failed to check stream %s: %w", streamName, err)
}

func (c *jetStreamClient) NewPublisher(streamName string) (Publisher, error) {
	if c.closed {
		return nil, ErrConnectionClosed
	}
	return &jsPublisher{js: c.js, stream: streamName}, nil
}

func (c *jetStreamClient) NewConsumer(streamName string, cfg ConsumerConfig) (Consumer, error) {
	if c.closed {
		return nil, ErrConnectionClosed
	}

	if cfg.Subject == "" {
		return nil, fmt.Errorf("%w: subject is required in ConsumerConfig", ErrInvalidConfig)
	}

	// Build subscription options
	opts := []nats.SubOpt{nats.BindStream(streamName)}

	if cfg.AckWait > 0 {
		opts = append(opts, nats.AckWait(cfg.AckWait))
	}
	if cfg.MaxDeliver > 0 {
		opts = append(opts, nats.MaxDeliver(cfg.MaxDeliver))
	}
	if cfg.MaxAckPending > 0 {
		opts = append(opts, nats.MaxAckPending(cfg.MaxAckPending))
	}

	sub, err := c.js.PullSubscribe(cfg.Subject, cfg.Durable, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Set defaults for required fields
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10 // default batch size
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 5 * time.Second // default batch timeout
	}

	return &jsConsumer{sub: sub, config: cfg}, nil
}

func (c *jetStreamClient) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	c.nc.Close()
	return nil
}
