package frizzle

import (
	"bytes"
)

// Transform is a function that modifies a Msg
type Transform func(Msg) Msg

// FrizTransformer provides a Transform to apply when a Msg is sent or received
type FrizTransformer interface {
	SendTransform() Transform
	ReceiveTransform() Transform
}

// WithTransformer returns an Option to add the provided FrizTransformer to a *Friz
func WithTransformer(ft FrizTransformer) Option {
	return func(f *Friz) {
		f.tforms = append(f.tforms, ft)
	}
}

var (
	_ FrizTransformer = (*SimpleSepTransformer)(nil)
)

// SimpleSepTransformer appends and removes a specified separator such as '\n' at the end of the Msg
type SimpleSepTransformer struct {
	sep []byte
}

// SendTransform returns a Transform to append the separator if it is not present at the end of Msg.Data()
func (st *SimpleSepTransformer) SendTransform() Transform {
	return func(m Msg) Msg {
		d := m.Data()
		if !bytes.Equal(d[len(d)-len(st.sep):len(d)], st.sep) {
			d = append(d, st.sep...)
		}
		return NewSimpleMsg(m.ID(), d, m.Timestamp())
	}
}

// ReceiveTransform returns a Transform to remove the separator if it is present at the end of Msg.Data()
func (st *SimpleSepTransformer) ReceiveTransform() Transform {
	return func(m Msg) Msg {
		d := m.Data()
		if bytes.Equal(d[len(d)-len(st.sep):len(d)], st.sep) {
			d = d[0 : len(d)-len(st.sep)]
		}
		return NewSimpleMsg(m.ID(), d, m.Timestamp())
	}
}

// NewSimpleSepTransformer initializes a new SepTransformer with a specified separator
func NewSimpleSepTransformer(sep []byte) FrizTransformer {
	return FrizTransformer(&SimpleSepTransformer{
		sep: sep,
	})
}
