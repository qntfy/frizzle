package basic

import (
	"github.com/qntfy/frizzle"
	"github.com/qntfy/frizzle/common"
	"github.com/spf13/viper"
)

var (
	_ frizzle.Source = (*Source)(nil)
)

const (
	// defaultBufferSize is 500 because that's the max number of kinesis records a lambda will return at once.
	defaultBufferSize = 500
)

// Source demonstrates the basic functionality of a Source
// (primarily for use in example code)
type Source struct {
	inChan   chan frizzle.Msg
	outChan  chan frizzle.Msg
	unAcked  *common.UnAcked
	failed   []frizzle.Msg
	isMock   bool
	quitChan chan struct{}
	doneChan chan struct{}
}

// InitSource initializes a basic Source
// populated with the string values in `basic_values` from
// provided viper. Close() will block until Ack() or Fail() has
// been called on all values.
func InitSource(config *viper.Viper) (*Source, chan<- frizzle.Msg, error) {
	bufferSize := defaultBufferSize
	if config.IsSet("buffer_size") {
		bufferSize = config.GetInt("buffer_size")
	}
	s := &Source{
		inChan:   make(chan frizzle.Msg),
		outChan:  make(chan frizzle.Msg, bufferSize),
		unAcked:  common.NewUnAcked(),
		isMock:   config.GetBool("mock"),
		quitChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
	if config.GetBool("track_fails") {
		s.failed = []frizzle.Msg{}
	}
	go s.consume()

	return s, s.Input(), nil
}

// consume tracks unacked Msgs while copying from inChan to outChan
func (s *Source) consume() {
	defer close(s.doneChan)
loop:
	for {
		select {
		case <-s.quitChan:
			break loop
		case m := <-s.inChan:
			s.unAcked.Add(m)
			s.outChan <- m
		}
	}
}

// Receive returns a channel to receive Msgs on
func (s *Source) Receive() <-chan frizzle.Msg {
	return (<-chan frizzle.Msg)(s.outChan)
}

// Input returns a channel to input Msgs on
func (s *Source) Input() chan<- frizzle.Msg {
	return (chan<- frizzle.Msg)(s.inChan)
}

// Ack acknowledges that processing is complete for the Msg
func (s *Source) Ack(m frizzle.Msg) error {
	if s.isMock {
		return nil
	}
	return s.unAcked.Remove(m)
}

// Fail reports the Msg as failed
func (s *Source) Fail(m frizzle.Msg) error {
	if s.isMock {
		return nil
	}
	err := s.unAcked.Remove(m)
	if err == nil && s.failed != nil {
		s.failed = append(s.failed, m)
	}
	return err
}

// Stop receiving Msgs
func (s *Source) Stop() error {
	// TODO: Currently this is not restart-able or checked by Close(). See #10
	close(s.quitChan)
	return nil
}

// Close errors if UnAcked Msgs remain
func (s *Source) Close() error {
	if s.isMock {
		return nil
	}
	if s.unAcked.Count() > 0 {
		return frizzle.ErrUnackedMsgsRemain
	}
	close(s.outChan)
	return nil
}

// Failed reports all Msgs which were Failed for the provided basic Source
// used for demonstrating functionality in the Example()
func (s *Source) Failed() []frizzle.Msg {
	return s.failed
}

// UnAcked reports all Msgs which are currently unAcked for the provided basic Source
// used for demonstrating functionality in the Example()
func (s *Source) UnAcked() []frizzle.Msg {
	return s.unAcked.List()
}

// UnAckedCount reports the count of UnAcked Msgs without generating a slice
func (s *Source) UnAckedCount() int {
	return s.unAcked.Count()
}
