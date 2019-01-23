package mocks

import frizzle "github.com/qntfy/frizzle"
import mock "github.com/stretchr/testify/mock"

// SinkEventer is a hand generated mock type for the Sink type and Eventer type
type SinkEventer struct {
	mock.Mock
}

// Close provides a mock function with given fields:
func (_m *SinkEventer) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Send provides a mock function with given fields: m, dest
func (_m *SinkEventer) Send(m frizzle.Msg, dest string) error {
	ret := _m.Called(m, dest)

	var r0 error
	if rf, ok := ret.Get(0).(func(frizzle.Msg, string) error); ok {
		r0 = rf(m, dest)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Events provides a mock function with given fields:
func (_m *SinkEventer) Events() <-chan frizzle.Event {
	ret := _m.Called()

	var r0 <-chan frizzle.Event
	if rf, ok := ret.Get(0).(func() <-chan frizzle.Event); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan frizzle.Event)
		}
	}

	return r0
}
