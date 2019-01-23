package basic

import (
	"github.com/qntfy/frizzle"
	"github.com/spf13/viper"
)

var (
	_ frizzle.Sink = (*Sink)(nil)
)

// Sink demonstrates the basic usage of a Sink (primarily for example code)
type Sink struct {
	sent   map[string][]frizzle.Msg
	isMock bool
}

// InitSink initializes a basic Sink that logs
// a map of what messages have been sent to which destinations.
func InitSink(v *viper.Viper) (*Sink, error) {
	return &Sink{
		sent:   make(map[string][]frizzle.Msg),
		isMock: v.GetBool("mock"),
	}, nil
}

// Send sends a Msg to provided dest
func (s *Sink) Send(m frizzle.Msg, dest string) error {
	if s.isMock {
		return nil
	}
	if _, ok := s.sent[dest]; !ok {
		s.sent[dest] = make([]frizzle.Msg, 0)
	}
	s.sent[dest] = append(s.sent[dest], m)
	return nil
}

// Close closes the Sink
func (s *Sink) Close() error {
	return nil
}

// Sent reports messages sent on a BasicSink
func (s *Sink) Sent(dest string) []frizzle.Msg {
	if _, ok := s.sent[dest]; !ok {
		return []frizzle.Msg{}
	}
	return s.sent[dest]
}
