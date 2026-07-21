package ui

import (
	"errors"
	"sync"
	"sync/atomic"
)

// ErrReported indicates that an error-level message was reported.
var ErrReported = errors.New("an error was reported")

// TransportError reports a failure to deliver an output message.
type TransportError struct {
	Err error
}

func (e *TransportError) Error() string { return "output delivery failed: " + e.Err.Error() }

// Unwrap returns the underlying transport error.
func (e *TransportError) Unwrap() error { return e.Err }

// Reporter classifies messages and records semantic and transport failures.
type Reporter struct {
	sink   Sink
	failed atomic.Bool
	mu     sync.Mutex
	errs   []error
}

// NewReporter returns a Reporter that delivers messages to sink.
func NewReporter(sink Sink) *Reporter { return &Reporter{sink: sink} }

// Ok reports an informational message.
func (r *Reporter) Ok(msg string) { r.write(Message{Level: LevelInfo, Text: msg}) }

// Warn reports a warning message.
func (r *Reporter) Warn(msg string) { r.write(Message{Level: LevelWarning, Text: msg}) }

// Err reports an error message and records a semantic failure.
func (r *Reporter) Err(msg string) {
	r.failed.Store(true)
	r.write(Message{Level: LevelError, Text: msg})
}

// EndMessage asks the sink to display its completion message.
func (r *Reporter) EndMessage() {
	if err := r.sink.EndMessage(); err != nil {
		r.record(err)
	}
}

func (r *Reporter) write(message Message) {
	if err := r.sink.Write(message); err != nil {
		r.record(err)
	}
}

func (r *Reporter) record(err error) {
	if err == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errs = append(r.errs, &TransportError{Err: err})
}

// Result returns all semantic and transport failures recorded so far.
func (r *Reporter) Result() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	errList := append([]error(nil), r.errs...)
	if r.failed.Load() {
		errList = append([]error{ErrReported}, errList...)
	}
	return errors.Join(errList...)
}
