package messaging

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/nats-io/nats.go"
)

type ConsumerConfig struct {
    // Subject is the primary subject this consumer will pull from
    Subject string

    // Subjects allows passing multiple subjects for reference/use by callers
    // (e.g., to ensure a stream). Not directly used by pull consumer creation.
    Subjects []string

    // FilterSubject sets the filter subject on the server-side consumer.
    // If empty, Subject is used.
    FilterSubject string

    // ConsumerGroup is a friendly alias for Durable when using pull consumers.
    // If Durable is empty and ConsumerGroup is set, it will be used as durable name.
    ConsumerGroup string

    // Durable is the durable consumer name
    Durable string

    // BatchSize controls how many messages to pull per fetch
    BatchSize int

    // BatchTimeout is the default timeout applied to Fetch calls when the
    // provided context has no deadline.
    BatchTimeout time.Duration

    // AckWait sets the server-side ack wait duration
    AckWait time.Duration

    // MaxDeliver configures the maximum redeliveries for a message
    MaxDeliver int

    // MaxAckPending caps the number of outstanding unacked messages
    MaxAckPending int
}

type Consumer interface {
	PullBatch(ctx context.Context) ([]*Message, error)
	Close() error
}

type jsConsumer struct {
    sub       *nats.Subscription
    batchSize    int
    batchTimeout time.Duration
}

func (c *jsConsumer) PullBatch(ctx context.Context) ([]*Message, error) {
    // Respect caller context; if no deadline and we have a default timeout,
    // apply it so Fetch does not block indefinitely.
    pullCtx := ctx
    if c.batchTimeout > 0 {
        if _, hasDeadline := ctx.Deadline(); !hasDeadline {
            var cancel context.CancelFunc
            pullCtx, cancel = context.WithTimeout(ctx, c.batchTimeout)
            defer cancel()
        }
    }

    msgs, err := c.sub.Fetch(c.batchSize, nats.Context(pullCtx))
    if err != nil {
        // If the fetch timed out or the context deadline was reached, we may
        // still have received some messages. In that case, return the partial
        // batch and suppress the timeout error. If we received no messages,
        // treat it as an empty batch rather than a hard error.
        if len(msgs) > 0 && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, nats.ErrTimeout)) {
            // proceed with msgs, clear error
            err = nil
        } else if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, nats.ErrTimeout) {
            // No messages available within the timeout window; return empty batch
            return nil, nil
        } else {
            return nil, err
        }
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

	return out, nil
}

func (c *jsConsumer) Close() error { return nil }
