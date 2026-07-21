package ui

import "errors"

type multiSink struct{ sinks []Sink }

// NewMultiSink returns a sink that delivers every call to every child sink.
func NewMultiSink(sinks ...Sink) Sink { return &multiSink{sinks: sinks} }

func (s *multiSink) Write(message Message) error {
	var errs []error
	for _, sink := range s.sinks {
		errs = append(errs, sink.Write(message))
	}
	return errors.Join(errs...)
}

func (s *multiSink) EndMessage() error {
	var errs []error
	for _, sink := range s.sinks {
		errs = append(errs, sink.EndMessage())
	}
	return errors.Join(errs...)
}

// MultiOutput fans every call out to all child outputs. It exists for
// contexts like `tpack init`, where stderr is invisible (run line in
// tmux.conf) and messages must also reach the tmux status line.
type MultiOutput struct {
	outputs []Output
}

// NewMultiOutput returns a MultiOutput wrapping the given outputs.
func NewMultiOutput(outputs ...Output) *MultiOutput {
	return &MultiOutput{outputs: outputs}
}

func (m *MultiOutput) Ok(msg string) {
	for _, o := range m.outputs {
		o.Ok(msg)
	}
}

func (m *MultiOutput) Warn(msg string) {
	for _, o := range m.outputs {
		o.Warn(msg)
	}
}

func (m *MultiOutput) Err(msg string) {
	for _, o := range m.outputs {
		o.Err(msg)
	}
}

func (m *MultiOutput) EndMessage() {
	for _, o := range m.outputs {
		o.EndMessage()
	}
}

// Result returns all failures recorded by the child outputs.
func (m *MultiOutput) Result() error {
	var errs []error
	for _, output := range m.outputs {
		errs = append(errs, output.Result())
	}
	return errors.Join(errs...)
}
