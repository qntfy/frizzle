// Code generated by mockery v1.0.0. DO NOT EDIT.
package mocks

import frizzle "github.com/qntfy/frizzle"
import mock "github.com/stretchr/testify/mock"

// Sink is an autogenerated mock type for the Sink type
type Sink struct {
	mock.Mock
}

// Close provides a mock function with given fields:
func (_m *Sink) Close() error {
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
func (_m *Sink) Send(m frizzle.Msg, dest string) error {
	ret := _m.Called(m, dest)

	var r0 error
	if rf, ok := ret.Get(0).(func(frizzle.Msg, string) error); ok {
		r0 = rf(m, dest)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}