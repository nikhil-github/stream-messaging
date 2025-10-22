package broker

import "errors"

// Common errors that can occur across different messaging implementations
var (
	// ErrStreamNotFound indicates the requested stream does not exist
	ErrStreamNotFound = errors.New("stream not found")

	// ErrInvalidConfig indicates the provided configuration is invalid
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrSubjectRequired indicates that a subject must be provided
	ErrSubjectRequired = errors.New("subject is required")

	// ErrNilMessage indicates an operation was attempted on a nil message
	ErrNilMessage = errors.New("nil message")

	// ErrConnectionClosed indicates the connection to the messaging system is closed
	ErrConnectionClosed = errors.New("connection closed")

	// ErrPublishFailed indicates a publish operation failed
	ErrPublishFailed = errors.New("publish failed")

	// ErrConsumerClosed indicates the consumer has been closed
	ErrConsumerClosed = errors.New("consumer closed")
)

