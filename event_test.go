package frizzle_test

import (
	"testing"

	"github.com/qntfy/frizzle"
	"github.com/stretchr/testify/assert"
)

type notEventer struct{}

func TestEventsNotImpl(t *testing.T) {
	_, ok := interface{}(&notEventer{}).(frizzle.Eventer)
	if ok {
		t.Error("notEventer should not implement frizzle.Eventer")
	}

	tests := [][]interface{}{
		{},
		{&notEventer{}},
		{(frizzle.Eventer)(nil)},
		{&notEventer{}, nil},
	}
	for _, test := range tests {
		ch := frizzle.InitEvents(test...)
		if _, ok := <-ch; ok {
			t.Errorf("InitEvents should return closed channel for: %v", t)
		}
	}
}

type eventer struct {
	evts chan frizzle.Event
}

func newEventer(size int, str string) *eventer {
	e := &eventer{
		evts: make(chan frizzle.Event, size),
	}
	for i := 0; i < size; i++ {
		e.evts <- frizzle.NewError(str)
	}
	close(e.evts)
	return e
}

func (e *eventer) Events() <-chan frizzle.Event {
	return (<-chan frizzle.Event)(e.evts)
}

var (
	_ frizzle.Eventer = (*eventer)(nil)
)

func TestEventsSimple(t *testing.T) {
	e := newEventer(1, "oh noes")

	ch := frizzle.InitEvents(e)
	rcv := make([]frizzle.Event, 0)
	for evt := range ch {
		rcv = append(rcv, evt)
	}

	assert.Equal(t, 1, len(rcv), "should be one event received")
	assert.Equal(t, "oh noes", rcv[0].String(), "output event should match input")
}

func TestEventsMulti(t *testing.T) {
	expected := map[string]int{
		"oh noes": 3,
		"failed":  2,
		"final":   7,
	}
	eventers := []interface{}{nil}
	for k, v := range expected {
		eventers = append(eventers, newEventer(v, k))
	}
	eventers = append(eventers, nil)

	ch := frizzle.InitEvents(eventers...)

	for evt := range ch {
		expected[evt.String()]--
	}

	for k, v := range expected {
		assert.Equalf(t, 0, v, "mismatch for key: %s", k)
	}
}
