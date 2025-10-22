package broker

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
)

type jsPublisher struct {
	js     nats.JetStreamContext
	stream string
}

func (p *jsPublisher) Publish(ctx context.Context, subject string, data []byte, opts *PublishOptions) error {
	if subject == "" {
		return ErrSubjectRequired
	}

	pubOpts := []nats.PubOpt{}

	// Add message ID if provided
	if opts != nil && opts.MessageID != "" {
		pubOpts = append(pubOpts, nats.MsgId(opts.MessageID))
	}

	// Ensure publish respects the provided context for timeout/cancel
	if ctx != nil {
		pubOpts = append(pubOpts, nats.Context(ctx))
	}

	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
	}

	// Add headers if provided
	if opts != nil && opts.Headers != nil {
		msg.Header = nats.Header{}
		for k, v := range opts.Headers {
			msg.Header.Set(k, v)
		}
	}

	_, err := p.js.PublishMsg(msg, pubOpts...)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPublishFailed, err)
	}

	return nil
}

func (p *jsPublisher) Close() error {
	return nil
}
