package frizzle

import (
	"sync"
)

// Event represents an async event from a Source or Sink
type Event interface {
	String() string
}

// Eventer is capable of reporting Events asynchronously through a channel
type Eventer interface {
	Events() <-chan Event
}

var (
	_ Event = (*Error)(nil)
	_ error = (*Error)(nil)
)

// Error conforms to Event and error interfaces for async error reporting
type Error struct {
	str string
}

// NewError creates an Error
func NewError(str string) *Error {
	return &Error{
		str: str,
	}
}

// String provides a string representation of the error
func (e *Error) String() string {
	return e.str
}

// Error provides a string version to conform to golang error interface
func (e *Error) Error() string {
	return e.String()
}

// InitEvents checks if objects are Eventers and merges any that are into one channel
// Note the returned channel will be closed immediately if none of the arguments are Eventers
// Exported in case of integrating events from multiple frizzles / sources / sinks
func InitEvents(ints ...interface{}) <-chan Event {
	chans := make([]<-chan Event, 0)
	for _, i := range ints {
		if eventer, ok := i.(Eventer); ok {
			chans = append(chans, eventer.Events())
		}
	}
	return mergeEvents(chans...)
}

// mergeEvents unifies multiple Events() channels
// from: https://gist.github.com/campoy/c19e2852b66c7d36cf28ac31b877c05b#file-main-go
func mergeEvents(cs ...<-chan Event) <-chan Event {
	out := make(chan Event)
	var wg sync.WaitGroup
	wg.Add(len(cs))
	for _, c := range cs {
		go func(c <-chan Event) {
			for v := range c {
				out <- v
			}
			wg.Done()
		}(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
