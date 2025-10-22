package messaging

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/nats-io/nats.go"
)

// ConsumerConfig controls how a JetStream consumer is created and how
// batches are fetched. Only a subset may be applicable depending on the
// server version and the underlying consumer type.
type ConsumerConfig struct {
    // Subject is the subject used when establishing the PullSubscribe. If
    // FilterSubject is set, it takes precedence for consumer creation and this
    // field may be ignored. Kept for backward compatibility.
    Subject string

    // Subjects is an optional list of subjects relevant to the consumer. This
    // is not used directly by PullSubscribe, but can be helpful to the caller
    // or future extensions that manage multi-filter consumers.
    Subjects []string

    // FilterSubject narrows the consumer to a single subject. If empty, the
    // Subject field is used.
    FilterSubject string

    // ConsumerGroup is a friendly name for load-sharing. If Durable is empty
    // we will use this as the durable name.
    ConsumerGroup string

    // Durable is the durable consumer name. If empty, ConsumerGroup will be
    // used when provided.
    Durable string

    // BatchSize is the number of messages to pull per fetch.
    BatchSize int

    // BatchTimeout bounds how long a PullBatch call should wait when there are
    // not enough messages available. If the incoming context already has a
    // deadline, it is respected over this value.
    BatchTimeout time.Duration

    // AckWait configures how long the server waits for an ack before
    // redelivering a message.
    AckWait time.Duration

    // MaxDeliver limits how many times a message may be redelivered before it
    // is sent to a DLQ (when configured) or dropped.
    MaxDeliver int

    // MaxAckPending sets the maximum number of unacknowledged messages the
    // server will allow for this consumer.
    MaxAckPending int
}

type Consumer interface {
	PullBatch(ctx context.Context) ([]*Message, error)
	Close() error
}

type jsConsumer struct {
    sub          *nats.Subscription
    batchSize    int
    batchTimeout time.Duration
}

func (c *jsConsumer) PullBatch(ctx context.Context) ([]*Message, error) {
    // Respect caller context; if no deadline and BatchTimeout is set, apply it.
    fetchCtx := ctx
    if c.batchTimeout > 0 {
        if _, hasDeadline := ctx.Deadline(); !hasDeadline {
            var cancel context.CancelFunc
            fetchCtx, cancel = context.WithTimeout(ctx, c.batchTimeout)
            defer cancel()
        }
    }

    msgs, err := c.sub.Fetch(c.batchSize, nats.Context(fetchCtx))
	if err != nil {
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

	return out, nil
}

func (c *jsConsumer) Close() error { return nil }

// ConsumeIntoChannel continuously pulls batches from the provided consumer and
// sends each message into the given channel until the context is done or a
// non-timeout error occurs. This does not ack messages; callers are responsible
// for acking after processing.
func ConsumeIntoChannel(ctx context.Context, consumer Consumer, out chan<- *Message) error {
    for {
        select {
        case <-ctx.Done():
            return nil
        default:
        }

        msgs, err := consumer.PullBatch(ctx)
        if err != nil {
            // Translate context timeouts into a short backoff and continue.
            if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
                // Yield to allow ctx cancellation to propagate.
                continue
            }
            return err
        }

        for _, m := range msgs {
            select {
            case <-ctx.Done():
                return nil
            case out <- m:
            }
        }
    }
}
