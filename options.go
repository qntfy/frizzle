package frizzle

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"go.uber.org/zap"
)

const (
	defaultFlushSeconds = 30
)

// Option is a type that modifies a Frizzle object
type Option func(*Friz)

// Logger specifies a zap.Logger for the Frizzle
func Logger(log *zap.Logger) Option {
	return func(f *Friz) {
		f.log = log
	}
}

// Stats specifies a stats object for the Frizzle
func Stats(stats StatsIncrementer) Option {
	return func(f *Friz) {
		f.stats = stats
	}
}

// FailSink specifies a Sink and dest to use on Fail for the Frizzle
func FailSink(s Sink, dest string) Option {
	return func(f *Friz) {
		f.failSink = s
		f.failDest = dest
		f.eventChan = InitEvents(f, s)
	}
}

// MonitorProcessingRate configures the Frizzle to periodically log the rate of Msgs processed.
func MonitorProcessingRate(pollPeriod time.Duration) Option {
	return func(f *Friz) {
		f.shutdownMonitor = make(chan struct{})
		go f.monitorProcessingRate(pollPeriod)
	}
}

// monitorProcessingRate implements the MonitorProcessingRate option by periodically logging the rate of Msgs Acked or Failed per second
func (f *Friz) monitorProcessingRate(pollPeriod time.Duration) {
	ticker := time.NewTicker(pollPeriod)
	for {
		select {
		case <-ticker.C:
			f.LogProcessingRate(pollPeriod)
		case <-f.shutdownMonitor:
			return
		}
	}
}

// LogProcessingRate implements the logic for MonitorProcessingRate and is exposed for testing.
func (f *Friz) LogProcessingRate(pollPeriod time.Duration) {
	current := atomic.LoadUint64(f.opsCount)
	currentRate := float64(current-f.lastRateCount) / pollPeriod.Seconds()
	f.log.Info("Processing Rate Update", zap.Float64("rate_per_sec", currentRate), zap.String("module", "monitor"))
	f.lastRateCount = current
}

// ReportAsyncErrors monitors Events() output and reports via logging and/or stats.
// error events are logged at Error level and have a stat recorded; all other
// events are logged at Warn level.
//
// If setting a FailSink, ReportAsyncErrors should be added/re-added AFTER the
// FailSink option or async events from the FailSink
func ReportAsyncErrors() Option {
	return func(f *Friz) {
		go f.ReportAsyncErrors()
	}
}

// ReportAsyncErrors monitors Events() output and reports via logging and/or stats
// It runs until f.Events() is closed and so should be run using the provided Option
// or in a separate goroutine.
func (f *Friz) ReportAsyncErrors() {
	for {
		// Ensure we call Events() with each iteration in case channel changes
		// e.g. due to new FailSink
		var evt Event
		var open bool
		if evt, open = <-f.Events(); !open {
			return
		}
		if err, ok := evt.(error); ok {
			f.log.Error(err.Error(), zap.Error(err))
			f.stats.Increment("ctr.error")
		} else {
			f.log.Warn(evt.String())
		}
	}
}

// setupSignalHandlers is used in HandleShutdown to monitor for SIGINT and SIGTERM.
func setupSignalHandlers() chan os.Signal {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, syscall.SIGTERM)
	return sigs
}

// HandleShutdown handles a clean shutdown for frizzle and calls an app provided shutdown function for
// SIGINT and SIGTERM. If Frizzle is run with this option, it does not need to call Close() explicitly
// as this is handled by HandleShutdown
func HandleShutdown(appShutdown func()) Option {
	return func(f *Friz) {
		sigs := setupSignalHandlers()
		go f.handleShutdown(sigs, appShutdown)
	}
}

// handleShutdown runs FlushAndClose() and provided appShutdown function whenever a signal is received.
func (f *Friz) handleShutdown(signals chan os.Signal, appShutdown func()) {
	for range signals {
		f.log.Warn("Shutdown: Pausing while Frizzle flushes", zap.Duration("maxPause", defaultFlushSeconds), zap.String("stage", "shutdown"))
		if err := f.FlushAndClose(defaultFlushSeconds * time.Second); err != nil {
			f.log.Error("Error in flushing and closing Frizzle", zap.Error(err), zap.String("stage", "shutdown"))
		}
		f.log.Debug("Frizzle flush complete, beginning app shutdown", zap.String("stage", "shutdown"))
		appShutdown()
		f.log.Warn("Shutdown complete", zap.String("stage", "shutdown"))
		return
	}
}
