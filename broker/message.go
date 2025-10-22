package broker

import (
	"github.com/nats-io/nats.go"
)

// Message represents a message received from a stream
type Message struct {
	// ID is a unique identifier for the message
	ID string

	// Subject is the subject/topic the message was published to
	Subject string

	// Data is the message payload
	Data []byte

	// Headers are key-value metadata attached to the message
	Headers map[string]string

	// Timestamp is the Unix timestamp when the message was created
	Timestamp int64

	// raw is the underlying NATS message for acknowledgement operations
	raw *nats.Msg
}

// Ack acknowledges successful processing of the message
func (m *Message) Ack() error {
	if m.raw == nil {
		return ErrNilMessage
	}
	return m.raw.Ack()
}

// Nak negatively acknowledges the message, signaling it should be redelivered
func (m *Message) Nak() error {
	if m.raw == nil {
		return ErrNilMessage
	}
	return m.raw.Nak()
}

// Term terminates the message, indicating it will never be processed successfully
func (m *Message) Term() error {
	if m.raw == nil {
		return ErrNilMessage
	}
	return m.raw.Term()
}

// InProgress signals to JetStream that the message is being processed
// and resets the AckWait timer on the server side.
func (m *Message) InProgress() error {
	if m.raw == nil {
		return ErrNilMessage
	}
	return m.raw.InProgress()
}

// AckSync acknowledges the message and waits for the server to confirm
// receipt of the acknowledgement before returning.
func (m *Message) AckSync() error {
	if m.raw == nil {
		return ErrNilMessage
	}
	return m.raw.AckSync()
}
