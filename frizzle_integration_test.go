package frizzle_test

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alexcesaro/statsd"
	"github.com/qntfy/frizzle"
	"github.com/qntfy/frizzle/basic"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Processor implements simple processing on a Frizzle
type Processor struct {
	frizzle.Frizzle
	count int
	quit  <-chan int
}

// Process() prints strings that are all lower case
// and keeps a running count of characters seen
func (p *Processor) Process(m frizzle.Msg) {
	data := m.Data()
	str := string(data)
	if str == "fail" {
		p.Fail(m)
		return
	}
	// count total characters seen
	p.count += len(str)
	// print and send any message that is only lower case
	if str == strings.ToLower(str) {
		fmt.Println(str)
		p.Send(m, "all-lower")
		p.Ack(m)
		return
	}
	// otherwise just Ack()
	p.Ack(m)
	return
}

// Loop processes received messages until quit signal received
func (p *Processor) Loop() {
	for {
		select {
		case <-p.quit:
			return
		case m := <-p.Receive():
			p.Process(m)
		}
	}
}

// Configure Viper for this example
func configViper() *viper.Viper {
	v := viper.GetViper()
	v.Set("track_fails", "true")
	return v
}

// helper method to extract payloads from []*msg.Msg
func msgsToStrings(msgs []frizzle.Msg) []string {
	result := make([]string, len(msgs))
	for i, m := range msgs {
		data := m.Data()
		result[i] = string(data)
	}
	return result
}

func inputMsgs(input chan<- frizzle.Msg, msgs []string) {
	for _, m := range msgs {
		input <- frizzle.NewSimpleMsg(m, []byte(m), time.Now())
	}
}

func Example() {
	// Initialize a Processor including a simple Frizzle message bus
	v := configViper()
	source, input, _ := basic.InitSource(v)
	lowerSink, _ := basic.InitSink(v)
	failSink, _ := basic.InitSink(v)
	exampleLog := exampleLogger()
	stats, _ := statsd.New(statsd.Mute(true))
	inputMsgs(input, []string{"foo", "BAR", "fail", "baSil", "frizzle"})

	f := frizzle.Init(source, lowerSink,
		frizzle.FailSink(failSink, "fail"),
		frizzle.Logger(exampleLog),
		frizzle.Stats(stats),
	)
	quit := make(chan int)
	p := &Processor{
		Frizzle: f,
		quit:    quit,
	}

	// Process messages
	go p.Loop()

	// Close() returns an error until all Msgs have Fail() or Ack() run
	stillRunning := true
	for stillRunning {
		select {
		case <-time.After(100 * time.Millisecond):
			if err := p.Close(); err == nil {
				stillRunning = false
			}
		}
	}
	f.(*frizzle.Friz).LogProcessingRate(1 * time.Second)
	exampleLog.Sync()
	quit <- 1

	// View results
	fmt.Printf("Characters seen: %d\n", p.count)
	fmt.Printf("Failed messages: %v\n", msgsToStrings(source.Failed()))
	fmt.Printf("Sent messages: %v\n", msgsToStrings(lowerSink.Sent("all-lower")))
	// Output:
	// foo
	// frizzle
	// {"level":"info","msg":"Processing Rate Update","rate_per_sec":5,"module":"monitor"}
	// Characters seen: 18
	// Failed messages: [fail]
	// Sent messages: [foo frizzle]
}

// exampleLogger replicates zap.NewExample() except at Info Level instead of Debug
func exampleLogger() *zap.Logger {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), os.Stdout, zap.InfoLevel)
	return zap.New(core)
}
