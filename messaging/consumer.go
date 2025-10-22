package messaging

import (
    "context"
    "fmt"
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

// ConsumerOptions defines server-side consumer configuration to ensure a durable.
// This is a subset commonly used in production; can be expanded as needed.
type ConsumerOptions struct {
    FilterSubject string
    AckWait       time.Duration
    MaxDeliver    int
    // Optional backoff schedule for redeliveries if MaxDeliver > 0
    Backoff       []time.Duration
}

type Consumer interface {
	PullBatch(ctx context.Context) ([]*Message, error)
	Close() error
}

type jsConsumer struct {
    sub       *nats.Subscription
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
        // Map headers (first value per key)
        hdrs := map[string]string{}
        if m.Header != nil {
            for k, vs := range m.Header {
                if len(vs) > 0 {
                    hdrs[k] = vs[0]
                }
            }
        }

        // Derive message ID from header or metadata
        id := ""
        if m.Header != nil {
            id = m.Header.Get("Nats-Msg-Id")
        }

        // Timestamp from JetStream metadata when available
        ts := time.Now().Unix()
        if meta, err := m.Metadata(); err == nil {
            if id == "" {
                id = fmt.Sprintf("%s:%d", meta.Stream, meta.Sequence.Stream)
            }
            if !meta.Timestamp.IsZero() {
                ts = meta.Timestamp.Unix()
            }
        }

        out = append(out, &Message{
            ID:        id,
            Subject:   m.Subject,
            Data:      m.Data,
            Headers:   hdrs,
            Timestamp: ts,
            raw:       m,
        })
    }

	if c.logger != nil {
		c.logger.Info("Fetched batch", "count", len(out))
	}
	return out, nil
}

func (c *jsConsumer) Close() error {
    if c.sub == nil {
        return nil
    }
    // Best-effort drain to allow in-flight acks to complete, else Unsubscribe
    if err := c.sub.Drain(); err != nil {
        // Fallback to Unsubscribe
        _ = c.sub.Unsubscribe()
    }
    return nil
}
