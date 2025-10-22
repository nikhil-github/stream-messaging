package stream1

import (
	"fmt"
	"github.com/nats-io/nats.go"
)

// publisher implements the Publisher interface
type publisher struct {
	js nats.JetStreamContext
}

// Publish publishes a message with additional options.
func (p *publisher) Publish(subject string, data []byte, opts PublishOptions) error {
	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
	}

	// Add message ID if provided
	if opts.MessageID != "" {
		msg.Header = nats.Header{}
		msg.Header.Set(nats.MsgIdHdr, opts.MessageID)
	}

	// Add custom headers
	if len(opts.Headers) > 0 {
		if msg.Header == nil {
			msg.Header = nats.Header{}
		}

		for key, value := range opts.Headers {
			msg.Header.Set(key, value)
		}
	}

	_, err := p.js.PublishMsg(msg)
	if err != nil {
		return fmt.Errorf("failed to publish message with options to subject %s: %w", subject, err)
	}

	return nil
}

// Close closes the publisher (no-op for now)
func (p *publisher) Close() error {
	return nil
}
