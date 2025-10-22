package broker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type jsConsumer struct {
	sub    *nats.Subscription
	config ConsumerConfig
	closed bool
}

func (c *jsConsumer) PullBatch(ctx context.Context) ([]*Message, error) {
	if c.closed {
		return nil, ErrConsumerClosed
	}

	// Create a context with batch timeout for this specific fetch operation
	pullCtx, cancel := context.WithTimeout(ctx, c.config.BatchTimeout)
	defer cancel()

	msgs, err := c.sub.Fetch(c.config.BatchSize, nats.Context(pullCtx))
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

	// Convert NATS messages to our Message type
	out := make([]*Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, c.convertMessage(m))
	}

	return out, nil
}

func (c *jsConsumer) Subscribe(ctx context.Context) (<-chan *Message, error) {
	if c.closed {
		return nil, ErrConsumerClosed
	}

	msgChan := make(chan *Message, c.config.BatchSize)

	go func() {
		defer close(msgChan)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msgs, err := c.sub.Fetch(c.config.BatchSize, nats.MaxWait(5*time.Second))
				if err != nil {
					// Timeout is expected when no messages are available
					if err == nats.ErrTimeout {
						continue
					}
					// For other errors, stop the subscription
					return
				}

				for _, m := range msgs {
					msg := c.convertMessage(m)
					select {
					case msgChan <- msg:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return msgChan, nil
}

func (c *jsConsumer) Close() error {
	c.closed = true
	return nil
}

// convertMessage converts a NATS message to our Message struct
func (c *jsConsumer) convertMessage(m *nats.Msg) *Message {
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

	return &Message{
		ID:        id,
		Subject:   m.Subject,
		Data:      m.Data,
		Headers:   hdrs,
		Timestamp: ts,
		raw:       m,
	}
}
