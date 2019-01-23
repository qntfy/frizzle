package frizzle_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/qntfy/frizzle"
	"github.com/qntfy/frizzle/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSimpleSep(t *testing.T) {
	var sep byte = '\n'
	updatedMsg := frizzle.NewSimpleMsg("updatedMsg", append(testMsgData, sep), time.Now())
	st := frizzle.NewSimpleSepTransformer([]byte{sep})

	// Send should append the sep character, same as updatedMsg
	sendResult := st.SendTransform()(testMsg).Data()
	assert.Equal(t, string(updatedMsg.Data()), string(sendResult))

	// Receive should remove the sep character so it matches original testMsg
	rcvResult := st.ReceiveTransform()(updatedMsg).Data()
	assert.Equal(t, string(testMsg.Data()), string(rcvResult))

	// Send should not update if sep character is already present
	sendResultNoUpdate := st.SendTransform()(updatedMsg).Data()
	assert.Equal(t, string(updatedMsg.Data()), string(sendResultNoUpdate))

	// Receive should not update if sep character is not present
	rcvResultNoUpdate := st.ReceiveTransform()(testMsg).Data()
	assert.Equal(t, string(testMsg.Data()), string(rcvResultNoUpdate))
}

func TestSimpleSepMultiChar(t *testing.T) {
	sep := []byte("end of file{}#")
	updatedMsg := frizzle.NewSimpleMsg("updatedMsg", append(testMsgData, sep...), time.Now())
	st := frizzle.NewSimpleSepTransformer(sep)

	// Send should append the sep slice, same as updatedMsg
	sendResult := st.SendTransform()(testMsg).Data()
	assert.Equal(t, string(updatedMsg.Data()), string(sendResult))

	// Receive should remove the sep slice so it matches original testMsg
	rcvResult := st.ReceiveTransform()(updatedMsg).Data()
	assert.Equal(t, string(testMsg.Data()), string(rcvResult))

	// Send should not update if sep character is already present
	sendResultNoUpdate := st.SendTransform()(updatedMsg).Data()
	assert.Equal(t, string(updatedMsg.Data()), string(sendResultNoUpdate))

	// Receive should not update if sep character is not present
	rcvResultNoUpdate := st.ReceiveTransform()(testMsg).Data()
	assert.Equal(t, string(testMsg.Data()), string(rcvResultNoUpdate))
}

func msgDataMatcher(data []byte) func(frizzle.Msg) bool {
	return func(m frizzle.Msg) bool {
		return bytes.Equal(data, m.Data())
	}
}

func TestSimpleSepSendWithFrizzle(t *testing.T) {
	f, _, mSink := testFrizzle()
	var sep byte = '\n'
	updatedMsg := frizzle.NewSimpleMsg("updatedMsg", append(testMsgData, sep), time.Now())
	st := frizzle.NewSimpleSepTransformer([]byte{sep})

	// With the transformer added, the final Send should include the updated data
	f.AddOptions(frizzle.WithTransformer(st))
	mSink.On("Send", mock.MatchedBy(msgDataMatcher(updatedMsg.Data())), testSinkName).Return(nil)

	err := f.Send(testMsg, testSinkName)
	assert.Nil(t, err)
	mSink.AssertExpectations(t)
}

func TestSimpleSepRcvWithFrizzle(t *testing.T) {
	mSource := &mocks.Source{}
	mSink := &mocks.Sink{}
	testChan := make(chan frizzle.Msg, 1)
	var testChanReceive <-chan frizzle.Msg
	testChanReceive = testChan
	mSource.On("Receive").Return(testChanReceive)
	f := frizzle.Init(mSource, mSink)

	// set up and attach the sep transformer
	var sep byte = '\n'
	updatedMsg := frizzle.NewSimpleMsg("updatedMsg", append(testMsgData, sep), time.Now())
	st := frizzle.NewSimpleSepTransformer([]byte{sep})
	f.AddOptions(frizzle.WithTransformer(st))

	// pass the message with the sep; the received message should be transformed and not have the sep
	testChan <- updatedMsg
	receivedMsg := <-f.Receive()
	assert.Equal(t, testMsgData, receivedMsg.Data())
	mSource.AssertExpectations(t)
}
