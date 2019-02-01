package frizzle_test

import (
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/qntfy/frizzle"
	"github.com/qntfy/frizzle/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var (
	testSinkName = "testSink"
	testMsgData  = []byte("some test data")
	testMsg      = frizzle.NewSimpleMsg("testMsg", testMsgData, time.Now())
	testLog, _   = zap.NewDevelopment()
)

var (
	_ frizzle.Sink   = (*mocks.Sink)(nil)
	_ frizzle.Source = (*mocks.Source)(nil)
)

// testFrizzle plugs in mocks and nops to a friz struct
func testFrizzle() (frizzle.Frizzle, *mocks.Source, *mocks.Sink) {
	mockSource := &mocks.Source{}
	mockSource.On("Receive").Return((<-chan frizzle.Msg)(make(chan frizzle.Msg))).Maybe()
	mockSink := &mocks.Sink{}
	f := frizzle.Init(mockSource, mockSink)
	return f, mockSource, mockSink
}

func TestBasicSend(t *testing.T) {
	f, _, mSink := testFrizzle()
	mSink.On("Send", testMsg, testSinkName).Return(nil)

	err := f.Send(testMsg, testSinkName)
	assert.Nil(t, err)
	mSink.AssertExpectations(t)
}

func TestNilSource(t *testing.T) {
	mSink := &mocks.Sink{}
	mSink.On("Send", testMsg, testSinkName).Return(nil)
	f := frizzle.Init(nil, mSink)

	err := f.Send(testMsg, testSinkName)
	assert.Nil(t, err)

	mSink.On("Close").Return(nil)
	err = f.Close()
	assert.Nil(t, err)
	mSink.AssertExpectations(t)
}

func TestBasicReceive(t *testing.T) {
	mSource := &mocks.Source{}
	mSink := &mocks.Sink{}
	testChan := make(chan frizzle.Msg, 1)
	var testChanReceive <-chan frizzle.Msg
	testChanReceive = testChan
	mSource.On("Receive").Return(testChanReceive)
	f := frizzle.Init(mSource, mSink)
	testChan <- testMsg

	receivedMsg := <-f.Receive()
	assert.Equal(t, testMsgData, receivedMsg.Data())
	mSource.AssertExpectations(t)
}

func TestNilSink(t *testing.T) {
	mSource := &mocks.Source{}
	testChan := make(chan frizzle.Msg, 1)
	var testChanReceive <-chan frizzle.Msg
	testChanReceive = testChan
	mSource.On("Receive").Return(testChanReceive)
	f := frizzle.Init(mSource, nil)
	testChan <- testMsg

	receivedMsg := <-f.Receive()
	assert.Equal(t, testMsgData, receivedMsg.Data())

	mSource.On("Close").Return(nil)
	err := f.Close()
	assert.Nil(t, err)
	mSource.AssertExpectations(t)
}

func TestBasicAck(t *testing.T) {
	f, mSource, _ := testFrizzle()
	mSource.On("Ack", testMsg).Return(nil)

	err := f.Ack(testMsg)
	assert.Nil(t, err)
	mSource.AssertExpectations(t)
}

func TestBasicFail(t *testing.T) {
	f, mSource, _ := testFrizzle()
	mSource.On("Fail", testMsg).Return(nil)

	err := f.Fail(testMsg)
	assert.Nil(t, err)
	mSource.AssertExpectations(t)
}

func TestBasicFailSink(t *testing.T) {
	f, mSource, _ := testFrizzle()
	failSink := &mocks.Sink{}
	failSink.On("Send", testMsg, "fail").Return(nil)
	f.AddOptions(frizzle.FailSink(failSink, "fail"))
	mSource.On("Fail", testMsg).Return(nil)

	err := f.Fail(testMsg)
	assert.Nil(t, err)
	mSource.AssertExpectations(t)
	failSink.AssertExpectations(t)
}

func TestBasicFlushAndClose(t *testing.T) {
	f, mSource, mSink := testFrizzle()
	mSource.On("Stop").Return(nil)
	mSource.On("UnAcked").Return([]frizzle.Msg{})
	mSource.On("Close").Return(nil)
	mSink.On("Close").Return(nil)

	err := f.FlushAndClose(10 * time.Millisecond)
	assert.Nil(t, err)
	mSource.AssertExpectations(t)
	mSink.AssertExpectations(t)
}

func TestHandleShutdown(t *testing.T) {
	f, mSource, mSink := testFrizzle()
	mSource.On("Stop").Return(nil).Once()
	mSource.On("UnAcked").Return([]frizzle.Msg{}).Once()
	mSource.On("Close").Return(nil).Once()
	mSink.On("Close").Return(nil).Once()

	shutdownChan := make(chan struct{})
	shutdown := func() {
		close(shutdownChan)
	}

	f.AddOptions(frizzle.HandleShutdown(shutdown))
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	<-shutdownChan
	mSource.AssertExpectations(t)
	mSink.AssertExpectations(t)
}

func newRecordedLogger() (*observer.ObservedLogs, *zap.Logger) {
	core, recorded := observer.New(zapcore.DebugLevel)
	zl := zap.New(core)
	return recorded, zl
}

func queueEventChan(evt frizzle.Event) <-chan frizzle.Event {
	testChanEvent := make(chan frizzle.Event, 1)
	testChanEvent <- evt
	close(testChanEvent)
	return (<-chan frizzle.Event)(testChanEvent)
}

type testEvt struct {
	str string
}

func (e *testEvt) String() string {
	return e.str
}

func TestReportAsyncErrorsLogError(t *testing.T) {
	mSource := &mocks.Source{}
	mSource.On("Receive").Return((<-chan frizzle.Msg)(make(chan frizzle.Msg))).Maybe()
	mEventer := &mocks.SinkEventer{}
	mEventer.On("Events").Return(queueEventChan(frizzle.NewError("oh noes"))).Once()
	f := frizzle.Init(mSource, mEventer).(*frizzle.Friz)

	logRecorder, log := newRecordedLogger()
	f.AddOptions(frizzle.Logger(log))
	f.ReportAsyncErrors()
	loggerEntry := logRecorder.All()[0]
	assert.Equal(t, "oh noes", loggerEntry.Message)
	assert.Equal(t, zapcore.ErrorLevel, loggerEntry.Level)
}

func TestReportAsyncErrorsStats(t *testing.T) {
	mSource := &mocks.Source{}
	mSource.On("Receive").Return((<-chan frizzle.Msg)(make(chan frizzle.Msg))).Maybe()
	mEventer := &mocks.SinkEventer{}
	mEventer.On("Events").Return(queueEventChan(frizzle.NewError("oh noes"))).Once()
	f := frizzle.Init(mSource, mEventer).(*frizzle.Friz)

	mStats := &mocks.StatsIncrementer{}
	mStats.On("Increment", "ctr.error").Return()
	f.AddOptions(frizzle.Stats(mStats))
	f.ReportAsyncErrors()
	mStats.AssertExpectations(t)
}

func TestReportAsyncErrorsWarnEvent(t *testing.T) {
	mSource := &mocks.Source{}
	mSource.On("Receive").Return((<-chan frizzle.Msg)(make(chan frizzle.Msg))).Maybe()
	mEventer := &mocks.SinkEventer{}
	mEventer.On("Events").Return(queueEventChan(&testEvt{str: "something happened"})).Once()
	f := frizzle.Init(mSource, mEventer).(*frizzle.Friz)

	logRecorder, log := newRecordedLogger()
	f.AddOptions(frizzle.Logger(log))
	f.ReportAsyncErrors()
	loggerEntry := logRecorder.All()[0]
	assert.Equal(t, "something happened", loggerEntry.Message)
	assert.Equal(t, zapcore.WarnLevel, loggerEntry.Level)
}

func TestReportAsyncErrorsBasicConcurrency(t *testing.T) {
	testChanEvent := make(chan frizzle.Event, 1)
	testChanEvent <- frizzle.NewError("nothing to see here")
	mSource := &mocks.Source{}
	mSource.On("Receive").Return((<-chan frizzle.Msg)(make(chan frizzle.Msg))).Maybe()
	mEventer := &mocks.SinkEventer{}
	mEventer.On("Events").Return((<-chan frizzle.Event)(testChanEvent)).Once()
	f := frizzle.Init(mSource, mEventer).(*frizzle.Friz)

	var wg sync.WaitGroup
	wg.Add(1)
	go func(f *frizzle.Friz) {
		defer wg.Done()
		f.ReportAsyncErrors()
	}(f)

	close(testChanEvent)
	wg.Wait()
}

func TestReportAsyncErrorsComplexConcurrency(t *testing.T) {
	testChanEvent := make(chan frizzle.Event, 1)
	mSource := &mocks.Source{}
	mSource.On("Receive").Return((<-chan frizzle.Msg)(make(chan frizzle.Msg))).Maybe()
	mEventer := &mocks.SinkEventer{}
	mEventer.On("Events").Return((<-chan frizzle.Event)(testChanEvent)).Once()
	logRecorder, log := newRecordedLogger()
	f := frizzle.Init(mSource, mEventer, frizzle.Logger(log)).(*frizzle.Friz)

	var wg sync.WaitGroup
	wg.Add(1)
	go func(f *frizzle.Friz) {
		defer wg.Done()
		f.ReportAsyncErrors()
	}(f)

	// ReportAsyncErrors should run and be waiting for input
	time.Sleep(5 * time.Millisecond)
	mEventer2 := &mocks.SinkEventer{}
	mEventer2.On("Events").Return(queueEventChan(frizzle.NewError("error1"))).Once()
	f.AddOptions(frizzle.FailSink(mEventer2, "fail"))

	// report the first event; next iteration should pick up the fail sink
	testChanEvent <- frizzle.NewError("error0")

	close(testChanEvent)
	wg.Wait()

	assert.Equal(t, "error0", logRecorder.All()[0].Message)
	assert.Equal(t, "error1", logRecorder.All()[1].Message)
}
