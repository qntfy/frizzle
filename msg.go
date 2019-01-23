package frizzle

import (
	"errors"
	"time"
)

var (
	// ErrAlreadyAcked is returned when Ack() or Fail() are called on a Msg that was already Acked or Failed
	ErrAlreadyAcked = errors.New("this Msg has already been Acked")
	// ErrUnackedMsgsRemain is returned when Source.Close() is called while len(Source.Unacked()) > 0
	ErrUnackedMsgsRemain = errors.New("attempting to close frizzle Source while there are still unAcked Msgs")
)

// Msg encapsulates an immutable message passed around by Frizzle
type Msg interface {
	ID() string
	Data() []byte
	Timestamp() time.Time
}

// Source defines a stream of incoming Msgs to be Received for processing,
// and reporting whether or not processing was successful.
type Source interface {
	Receive() <-chan Msg
	Ack(m Msg) error
	Fail(m Msg) error
	UnAcked() []Msg
	Stop() error
	Close() error
}

// Sink defines a message service where Msgs can be sent as part of processing.
type Sink interface {
	Send(m Msg, dest string) error
	Close() error
}

// SimpleMsg is a basic Msg implementation
type SimpleMsg struct {
	id        string
	data      []byte
	timestamp time.Time
}

// ID returns the ID
func (s *SimpleMsg) ID() string {
	return s.id
}

// Data returns the Data
func (s *SimpleMsg) Data() []byte {
	return s.data
}

// Timestamp returns the Timestamp
func (s *SimpleMsg) Timestamp() time.Time {
	return s.timestamp
}

// NewSimpleMsg creates a new SimpleMsg
func NewSimpleMsg(id string, data []byte, timestamp time.Time) Msg {
	m := &SimpleMsg{
		id:        id,
		data:      data,
		timestamp: timestamp,
	}
	return Msg(m)
}
