package stream1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// consumer implements the Consumer interface
type consumer struct {
	js         nats.JetStreamContext
	streamName string
	config     ConsumerConfig
}

// PullBatch pulls a batch of messages and returns them via a channel
func (c *consumer) PullBatch(ctx context.Context) (<-chan Message, error) {
	// Create or get existing consumer
	consumerName := c.config.ConsumerGroup
	if consumerName == "" {
		consumerName = fmt.Sprintf("%s-consumer", c.streamName)
	}

	// Check if consumer exists, create if not
	_, err := c.js.ConsumerInfo(c.streamName, consumerName)
	if err != nil {
		if strings.Contains(err.Error(), "consumer not found") {
			consumerConfig := &nats.ConsumerConfig{
				Durable:       consumerName,
				DeliverPolicy: nats.DeliverAllPolicy,
				AckPolicy:     nats.AckExplicitPolicy,
				FilterSubject: c.config.FilterSubject,
				ReplayPolicy:  nats.ReplayInstantPolicy,
				MaxDeliver:    c.config.MaxDeliver,
				MaxAckPending: c.config.MaxAckPending,
			}
			_, err = c.js.AddConsumer(c.streamName, consumerConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create consumer %s: %w", consumerName, err)
			}
		} else {
			return nil, fmt.Errorf("failed to check consumer %s: %w", consumerName, err)
		}
	}

	// Create pull subscription with filter subject
	filterSubject := c.config.FilterSubject
	if filterSubject == "" && len(c.config.Subjects) > 0 {
		filterSubject = c.config.Subjects[0] // Use first subject as filter
	}

	sub, err := c.js.PullSubscribe(filterSubject, consumerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull subscription: %w", err)
	}

	// Create message channel
	msgChan := make(chan Message, c.config.BatchSize)

	// Start goroutine to pull messages
	go func() {
		defer close(msgChan)
		defer sub.Unsubscribe()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Pull batch of messages
				msgs, err := sub.Fetch(c.config.BatchSize, nats.MaxWait(c.config.BatchTimeout))
				if err != nil {
					if err == nats.ErrTimeout {
						continue // No messages available, try again
					}
					return // Other error, stop pulling
				}

				// Send each message to the channel
				for _, msg := range msgs {
					select {
					case msgChan <- &message{msg: msg}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return msgChan, nil
}

// Close closes the consumer (no-op for now)
func (c *consumer) Close() error {
	return nil
}

// message implements the Message interface
type message struct {
	msg *nats.Msg
}

// Subject returns the message subject
func (m *message) Subject() string {
	return m.msg.Subject
}

// Data returns the message payload
func (m *message) Data() []byte {
	return m.msg.Data
}

// Headers returns the message headers
func (m *message) Headers() map[string]string {
	headers := make(map[string]string)
	for key, values := range m.msg.Header {
		if len(values) > 0 {
			headers[key] = values[0] // Take first value
		}
	}
	return headers
}

// MessageID returns a unique message ID
func (m *message) MessageID() string {
	// Use NATS message subject + data hash as ID
	return fmt.Sprintf("%s-%x", m.msg.Subject, m.msg.Data)
}

// Timestamp returns the message timestamp
func (m *message) Timestamp() time.Time {
	// NATS doesn't provide timestamp, use current time
	return time.Now()
}

// Ack acknowledges the message
func (m *message) Ack() error {
	return m.msg.Ack()
}

// Nack negatively acknowledges the message
func (m *message) Nack() error {
	return m.msg.Nak()
}
