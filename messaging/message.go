package messaging

import (
    "errors"

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

// InProgress signals to JetStream that the message is being processed
// and resets the AckWait timer on the server side.
func (m *Message) InProgress() error {
    if m.raw == nil {
        return errors.New("nil message")
    }
    return m.raw.InProgress()
}

// AckSync acknowledges the message and waits for the server to confirm
// receipt of the acknowledgement before returning.
func (m *Message) AckSync() error {
    if m.raw == nil {
        return errors.New("nil message")
    }
    return m.raw.AckSync()
}
