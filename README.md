# Frizzle

[![Travis Build Status](https://img.shields.io/travis/qntfy/frizzle.svg?branch=master)](https://travis-ci.org/qntfy/frizzle)
[![Coverage Status](https://coveralls.io/repos/github/qntfy/frizzle/badge.svg?branch=master)](https://coveralls.io/github/qntfy/frizzle?branch=master)
[![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
[![GitHub release](https://img.shields.io/github/release/qntfy/frizzle.svg?maxAge=3600)](https://github.com/qntfy/frizzle/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/qntfy/frizzle)](https://goreportcard.com/report/github.com/qntfy/frizzle)
[![GoDoc](https://godoc.org/github.com/qntfy/frizzle?status.svg)](http://godoc.org/github.com/qntfy/frizzle)

Frizzle is a magic message (`Msg`) bus designed for parallel processing w many goroutines.
  * `Receive()` messages from a configured `Source`
  * Do your processing, possibly `Send()` each `Msg` on to one or more `Sink` destinations
  * `Ack()` (or `Fail()`) the `Msg`  to notify the `Source` that processing completed

## Getting Started

**Start with the [example implementation](frizzle_integration_test.go)** which shows a simple canonical
implementation of a `Processor` on top of Frizzle and most of the core functions.

high level interface
```
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

func Init(source Source, sink Sink, opts ...Option) Frizzle
```

The core of the repo is a `Friz` struct (returned by `Init()`) which implements `Frizzle`. The intent is for
separate `Source` and `Sink` implementations (in separate repos) to be mixed and matched with the glue of
`Frizzle`. A processing library can take a `Frizzle` input to allow easy re-use with multiple
underlying message technologies.
Friz also implements `Source` and `Sink` to allow chaining if needed.

### Source and Sink Implementations
* **[Frinesis](https://github.com/qntfy/frinesis)** for AWS Kinesis
* **[Frafka](https://github.com/qntfy/frafka)** for Apache Kafka
* **[Basic](./basic)** for simple usage (part of Frizzle)

If you write a new implementation, we'd love to add it to our list!

## Msg
A basic interface which can be extended:
```
// Msg encapsulates an immutable message passed around by Frizzle
type Msg interface {
	ID() string
	Data() []byte
	Timestamp() time.Time
}
```

A `frizzle.SimpleMsg` struct is provided for basic use cases.

## Source and Sink

```
// Source defines a stream of incoming Msgs to be Received for processing,
// and reporting whether or not processing was successful.
type Source interface {
	Receive() <-chan Msg
	Ack(m Msg) error
	Fail(m Msg) error
	UnAcked() []Msg
	Stop() error
	Close() error
}

// Sink defines a message service where Msgs can be sent as part of processing.
type Sink interface {
	Send(m Msg, dest string) error
	Close() error
}
```

## Options
Frizzle supports a variety of `Option` parameters for additional functionality to simplify your integration.
These can be included with `Init()` or added using a `friz.AddOptions()` call. Note that `AddOptions()`
updates the current friz and does not return anything.

Currently supported options:
* `Logger(log *zap.Logger)` - Include a logger to report frizzle-internal logging.
* `Stats(stats StatsIncrementer)` - Include a stats client for frizzle-internal metrics reporting.
  See [Stats](#stats) for what metrics are supported.
* `FailSink(s Sink, dest string)` - Provide a Sink and destination (kafka topic, kinesis stream etc)
  where `Fail()`ed Msgs will be sent automatically.
* `MonitorProcessingRate(pollPeriod time.Duration)` - Log the sum count of Acked and Failed Msgs every `pollPeriod`.
* `ReportAsyncErrors()` - Launch a simple go routine to monitor the `Events()` channel. All events are logged at `Error` or `Warn` level;
  any events that match `error` interface have a stat recorded. Logging and/or stats are disabled
  if `Logger()`/`Stats()` have not been set, respectively.
  * This is a most basic handling that does not account for any specific Event types from Source/Sink implementations;
  developers should write an app specific monitoring routine to parse and handle specific Event cases
  (for which this can be a helpful starting template).
* `HandleShutdown(appShutdown func())` - Monitor for `SIGINT` and `SIGTERM`, call `FlushAndClose()` followed by
  provided `appShutdown` when they are received.
* `WithTransformer(ft FrizTransformer)` - Add a transformer to modify the Msg's before they are sent or received.
  Currently only supports a "Simple Separator" Transformer which adds a specified record separator (such as newline)
  before sending if it isn't already present, and removes the same separator on receive if it is present.

## Events
Since Source and Sink implementations often send and receive Msgs in batch fashion,
They often may find out about any errors (or other important events) asynchronously.
To support this, async events can be recovered via a channel returned by the `Friz.Events()` method.
If a Source/Sink does not implement the `Eventer` interface this functionality will be ignored.

### Caveats for using `Events()`

* Frizzle Events must provide a minimum `String()` interface; when consuming Events
a type assertion switch is highly recommended to receive other relevant information.
  * **A `default:` trap for unhandled cases is also highly recommended!**
  * For a reference implementation of the same interface see
    **[here](https://github.com/confluentinc/confluent-kafka-go/blob/d87f439f4a3ac1a1f94f3071ce2ed2238e27fba4/examples/producer_channel_example/producer_channel_example.go#L48-L66)**
* A Friz's `Events()` channel will be closed after all underlying Source/Sink `Events()` channels are closed.
  * If a Friz is initialized without any Source/Sinks that implement `Events()`, the channel returned by
    `Friz.Events()` will be closed immediately.

In addition to the `String()` method required by frizzle, currently only errors are
returned by frinesis (no other event types) so all Events recovered will also conform
to `error` interface.

## Transformers
Transformers provide a mechanism to do simple updates to a `Msg` prior to a `Send()` or `Receive()`, which 
can be added at initializiation but is otherwise transparent to the processor and Source/Sink.
This can be useful in a case where e.g. you need to apply a transform when running on one messaging platform
but not another, and don't want to expose the core processing code to information about which platform
is in use.

Frizzle supports adding Transformers with a `WithTransformer()` Option:
```
// WithTransformer returns an Option to add the provided FrizTransformer to a Frizzle
func WithTransformer(ft FrizTransformer) Option

// Transform is a type that modifies a Msg
type Transform func(Msg) Msg

// FrizTransformer provides a Transform to apply when a Msg is sent or received
type FrizTransformer interface {
	SendTransform() Transform
	ReceiveTransform() Transform
}

```

An example implementation to add and remove a separator suffix on each Msg is included in
[transform.go](./transform.go). To reduce clutter we generally suggest implementing a new
Transform in a separate repo, but we can consider adding high utility ones here.

## Prereqs / Build instructions

### Go mod

As of Go 1.11, frizzle uses [go mod](https://github.com/golang/go/wiki/Modules) for dependency management.

### Install

```
$ go get github.com/qntfy/frizzle
$ cd frizzle
$ go build
```

## Running the tests

`go test -v --cover ./...`

## Configuration
We recommend building Sources and Sinks to initialize using [Viper](https://godoc.org/github.com/spf13/viper),
typically through environment variables (but client can do whatever it wants, just needs to provide the
configured Viper object with relevant values). The application might use a prefix such as before the below values.

### Basic
| Variable | Required | Description | Default |
|---------------------------|:--------:|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-------:|
| BUFFER_SIZE | source (optional) | size of `Input()` channel buffer | 500 |
| MOCK | optional | mocks don't track sent or unacked Msgs, just return without error | false |

## Stats
`StatsIncrementer` is a simple interface with just `Increment(bucket string)`; based on `github.com/alexcesaro/statsd` 
but potentially compatible with a variety of metrics engines. When `Stats()` is set, Frizzle records the following metrics. 
If a `Logger()` has been set, each of the below also generates a Debug level log with the ID() of the Msg.

| Bucket | Description |
|---------------------------|--------------------------------------------------|
| ctr.rcv | count of Msgs received from Source |
| ctr.send | count of Msgs sent to Sink |
| ctr.ack | count of Msgs Ack'ed by application |
| ctr.fail | count of Msgs Fail'ed by application |
| ctr.failsink | count of Msgs sent to FailSink |
| ctr.error | count of `error`s from `Events()`* |

\* only recorded if ReportAsyncErrors is running

## Contributing
Contributions welcome! Take a look at open issues. New Source/Sink implementations should be added in separate repos.
If you let us know (and link to test demonstrating it conforms to the interface) we are happy to link them here!
