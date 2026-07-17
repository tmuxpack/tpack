package ui

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

// HasFailed reports whether any child output has failed.
func (m *MultiOutput) HasFailed() bool {
	for _, o := range m.outputs {
		if o.HasFailed() {
			return true
		}
	}
	return false
}
