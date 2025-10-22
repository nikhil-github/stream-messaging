package messaging
package messaging

import (
	"context"
	"errors"
	"log/slog"

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
	logger *slog.Logger
}

func (p *jsPublisher) Publish(ctx context.Context, data []byte, opts *PublishOptions) error {
	if opts == nil || opts.Subject == "" {
		return errors.New("subject required")
	}

	pubOpts := []nats.PubOpt{}
	if opts.MessageID != "" {
		pubOpts = append(pubOpts, nats.MsgId(opts.MessageID))
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
		if p.logger != nil {
			p.logger.Warn("Failed to publish", "subject", opts.Subject, "error", err)
		}
		return err
	}

	if p.logger != nil {
		p.logger.Info("Published message", "subject", opts.Subject, "seq", pa.Sequence)
	}
	return nil
}

func (p *jsPublisher) Close() error {
	return nil
}
