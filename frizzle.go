package frizzle

import (
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Frizzle is a Msg bus for rapidly configuring and processing messages between multiple message services.
type Frizzle interface {
	Receive() <-chan Msg
	Send(m Msg, dest string) error
	Ack(Msg) error
	Fail(Msg) error
	Events() <-chan Event
	AddOptions(...Option)
	FlushAndClose(timeout time.Duration) error
	Close() error
}

var (
	_ Frizzle = (*Friz)(nil)
)

// Type identifies the supported types of Frizzle Sources and Sinks for use in dependent repos
type Type string

const (
	// Kafka (Apache: http://kafka.apache.org/)
	Kafka Type = "kafka"
	// Kinesis (AWS: https://aws.amazon.com/kinesis/)
	Kinesis Type = "kinesis"
)

// Friz is the internal struct implementing Frizzle.
type Friz struct {
	source          Source
	sink            Sink
	log             *zap.Logger
	stats           StatsIncrementer
	failSink        Sink
	failDest        string
	opsCount        *uint64
	lastRateCount   uint64
	shutdownMonitor chan struct{}
	tforms          []FrizTransformer
	outChan         chan Msg
	eventChan       <-chan Event
}

// Init takes an initialized Source and Sink, and a set of Options.
// It returns a configured Frizzle.
func Init(source Source, sink Sink, opts ...Option) Frizzle {
	var ops uint64
	f := &Friz{
		source:   source,
		sink:     sink,
		log:      zap.NewNop(),
		stats:    &noopIncrementer{},
		opsCount: &ops,
		tforms:   []FrizTransformer{},
		outChan:  make(chan Msg),
	}

	// apply all passed in configuration fx
	for _, opt := range opts {
		opt(f)
	}

	if f.source != nil {
		go f.consume()
	}

	f.eventChan = InitEvents(f.source, f.sink)

	return Frizzle(f)
}

// AddOptions configures the Frizzle with the supplied Options.
func (f *Friz) AddOptions(opts ...Option) {
	// apply all passed in configuration fx
	for _, opt := range opts {
		opt(f)
	}
}

// Events returns the async Event channel
// Note if neither Source or Sink implement Events(), it will be closed
// immediately on init.
func (f *Friz) Events() <-chan Event {
	return f.eventChan
}

// consume receives Msgs from the Source and passes them to the outChan
func (f *Friz) consume() {
	for m := range f.source.Receive() {
		// Apply any Transforms which have been added
		for _, tform := range f.tforms {
			m = tform.ReceiveTransform()(m)
		}
		f.stats.Increment("ctr.rcv")
		f.log.Debug(fmt.Sprintf("Received a message: %s", m.ID()), zap.String("ID", m.ID()))
		f.outChan <- m
	}
}

// Receive a receiving channel to get incoming Msgs from the Source.
func (f *Friz) Receive() <-chan Msg {
	return (<-chan Msg)(f.outChan)
}

// Send the Msg to Sink identified by sinkName
func (f *Friz) Send(m Msg, dest string) error {
	f.stats.Increment("ctr.send")
	f.log.Debug(fmt.Sprintf("Sending a message '%s' to dest %s", m.ID(), dest), zap.String("ID", m.ID()), zap.String("dest", dest))
	// Apply any Transforms which have been added
	for _, tform := range f.tforms {
		m = tform.SendTransform()(m)
	}
	return f.sink.Send(m, dest)
}

// Ack reports to the Source that processing completed successfully for this Msg
func (f *Friz) Ack(m Msg) error {
	f.stats.Increment("ctr.ack")
	f.log.Debug(fmt.Sprintf("Received Ack for message %s", m.ID()), zap.String("ID", m.ID()))
	atomic.AddUint64(f.opsCount, 1)
	return f.source.Ack(m)
}

// Fail reports to the Source that processing failed for this Msg, and optionally sends to a Fail-specific Sink
func (f *Friz) Fail(m Msg) error {
	f.stats.Increment("ctr.fail")
	f.log.Debug(fmt.Sprintf("Received Fail for message %s", m.ID()), zap.String("ID", m.ID()))
	atomic.AddUint64(f.opsCount, 1)
	if err := f.source.Fail(m); err != nil {
		return err
	}
	if f.failSink != nil && f.failDest != "" {
		f.stats.Increment("ctr.failsend")
		f.log.Debug(fmt.Sprintf("Sending a message '%s' to fail sink '%s'", m.ID(), f.failDest),
			zap.String("ID", m.ID()), zap.String("dest", f.failDest))
		if err := f.failSink.Send(m, f.failDest); err != nil {
			return err
		}
	}
	return nil
}

// FlushAndClose provides default logic for stopping, emptying and shutting down
// the configured Source and Sink. Any Msgs which are still unAcked after the
// timeout has expired are Failed.
func (f *Friz) FlushAndClose(timeout time.Duration) error {
	if f.source == nil {
		panic("FlushAndClose() should not be called without a Source configured; use Close()")
	}
	f.source.Stop()
	tick := time.NewTicker(timeout / 10).C
	timeoutAlarm := time.After(timeout)
	for len(f.source.UnAcked()) > 0 {
		select {
		case <-tick: // periodically check if len(UnAcked()) == 0
		case <-timeoutAlarm:
			// fail out remaining messages if timeout expires
			unAcked := f.source.UnAcked()
			f.log.Warn("Flush timed out with messages still remaining",
				zap.Int("remaining_msgs", len(unAcked)))
			for _, m := range unAcked {
				if err := f.Fail(m); err != nil && err != ErrAlreadyAcked {
					return err
				}
			}
			f.log.Warn("All timed out messages have been Failed")
		}
	}
	return f.Close()
}

// Close down the Frizzle, the Source and all configured Sinks gracefully.
// The Frizzle must not be used afterward.
func (f *Friz) Close() error {
	f.log.Debug("Attempting to Close frizzle")
	if f.sink != nil {
		if err := f.sink.Close(); err != nil {
			return err
		}
	}
	if f.source != nil {
		if err := f.source.Close(); err != nil {
			return err
		}
	}
	if f.failSink != nil {
		if err := f.failSink.Close(); err != nil {
			return err
		}
	}

	// stop f.monitorProcessingRate if it is running
	if f.shutdownMonitor != nil {
		close(f.shutdownMonitor)
		f.shutdownMonitor = nil
	}

	f.log.Debug("Closing frizzle successful")
	return nil
}
