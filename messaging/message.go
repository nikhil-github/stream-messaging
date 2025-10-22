package messaging

import (
	"errors"
	"time"

	"github.com/nats-io/nats.go"
)

type Message struct {
	ID        string
	Subject   string
	Data      []byte
	Headers   map[string]string
	Timestamp int64
	raw       *nats.Msg
}

func (m *Message) Ack() error {
	if m.raw == nil {
		return errors.New("nil message")
	}
	return m.raw.Ack()
}

func (m *Message) Nak() error {
	if m.raw == nil {
		return errors.New("nil message")
	}
	return m.raw.Nak()
}

func (m *Message) Term() error {
	if m.raw == nil {
		return errors.New("nil message")
	}
	return m.raw.Term()
}
