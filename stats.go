package frizzle

// StatsIncrementer is a simple stats interface that supports incrementing a bucket
// Met by github.com/alexcesaro/statsd and similar; used for mocking and multiple impls
type StatsIncrementer interface {
	Increment(bucket string)
}

var (
	_ StatsIncrementer = (*noopIncrementer)(nil)
)

// noopIncrementer is a NoOp implementation of StatsIncrementer
type noopIncrementer struct{}

// Increment is a Noop to meet the StatsIncrementer interfacee
func (n *noopIncrementer) Increment(bucket string) {
	return
}
