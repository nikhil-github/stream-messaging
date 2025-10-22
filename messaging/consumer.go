package messaging

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

type ConsumerConfig struct {
	Subject   string
	BatchSize int
	AckWait   time.Duration
	Durable   string
}

type Consumer interface {
	PullBatch(ctx context.Context) ([]*Message, error)
	Close() error
}

type jsConsumer struct {
	sub       *nats.PullSubscription
	batchSize int
	logger    *slog.Logger
}

func (c *jsConsumer) PullBatch(ctx context.Context) ([]*Message, error) {
	msgs, err := c.sub.Fetch(c.batchSize, nats.Context(ctx))
	if err != nil {
		if c.logger != nil {
			c.logger.Warn("Failed to fetch batch", "error", err)
		}
		return nil, err
	}

	out := make([]*Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, &Message{
			Subject:   m.Subject,
			Data:      m.Data,
			Headers:   map[string]string{},
			Timestamp: time.Now().Unix(),
			raw:       m,
		})
	}

	if c.logger != nil {
		c.logger.Info("Fetched batch", "count", len(out))
	}
	return out, nil
}

func (c *jsConsumer) Close() error { return nil }
