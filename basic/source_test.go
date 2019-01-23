package basic_test

import (
	"testing"
	"time"

	"github.com/qntfy/frizzle"
	"github.com/qntfy/frizzle/basic"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var (
	testMsgData = "some frizzle data"
	testMsg     = frizzle.NewSimpleMsg("1", []byte(testMsgData), time.Now())
)

func testSource() (frizzle.Source, chan<- frizzle.Msg) {
	src, input, _ := basic.InitSource(viper.New())
	input <- testMsg
	return src, input
}

func TestBasicReceive(t *testing.T) {
	src, _ := testSource()

	receivedMsg := <-src.Receive()
	assert.Equal(t, testMsgData, string(receivedMsg.Data()))
}

func TestBasicAck(t *testing.T) {
	src, _ := testSource()

	<-src.Receive()
	assert.Equal(t, []frizzle.Msg{testMsg}, src.UnAcked())
	err := src.Ack(testMsg)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(src.UnAcked()))
}

func TestBasicFail(t *testing.T) {
	src, _ := testSource()

	<-src.Receive()
	assert.Equal(t, []frizzle.Msg{testMsg}, src.UnAcked())
	err := src.Fail(testMsg)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(src.UnAcked()))
}

func TestBasicClose(t *testing.T) {
	src, _ := testSource()
	assert.Equal(t, frizzle.ErrUnackedMsgsRemain, src.Close())

	<-src.Receive()
	assert.Equal(t, frizzle.ErrUnackedMsgsRemain, src.Close())

	src.Ack(testMsg)
	assert.Nil(t, src.Close())
}
