package messaging

import (
	"context"
	"errors"

	"github.com/nats-io/nats.go"
)

type PublishOptions struct {
	MessageID string
	Headers   map[string]string
	Subject   string // required
}

type Publisher interface {
	Publish(ctx context.Context, data []byte, opts *PublishOptions) error
	Close() error
}

type jsPublisher struct {
	js     nats.JetStreamContext
	stream string
}

func (p *jsPublisher) Publish(ctx context.Context, data []byte, opts *PublishOptions) error {
	if opts == nil || opts.Subject == "" {
		return errors.New("subject required")
	}

	pubOpts := []nats.PubOpt{}
	if opts.MessageID != "" {
		pubOpts = append(pubOpts, nats.MsgId(opts.MessageID))
	}
    // Ensure publish respects the provided context for timeout/cancel
    if ctx != nil {
        pubOpts = append(pubOpts, nats.Context(ctx))
    }

	msg := &nats.Msg{
		Subject: opts.Subject,
		Data:    data,
	}
	if opts.Headers != nil {
		msg.Header = nats.Header{}
		for k, v := range opts.Headers {
			msg.Header.Set(k, v)
		}
	}

    pa, err := p.js.PublishMsg(msg, pubOpts...)
	if err != nil {
		return err
	}

    _ = pa
	return nil
}

func (p *jsPublisher) Close() error {
	return nil
}
