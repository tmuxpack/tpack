package ui

type displayMessenger interface {
	DisplayMessage(string) error
}

type statusSink struct{ runner displayMessenger }

// NewStatusSink returns a sink that displays messages in the tmux status line.
func NewStatusSink(runner displayMessenger) Sink {
	return &statusSink{runner: runner}
}

func (s *statusSink) Write(message Message) error {
	prefix := "tpack: "
	switch message.Level {
	case LevelWarning:
		prefix = "tpack: warning: "
	case LevelError:
		prefix = "tpack: error: "
	case LevelInfo:
	default:
	}
	return s.runner.DisplayMessage(tmuxLiteral(prefix + message.Text))
}

func (s *statusSink) EndMessage() error { return nil }

// NewStatusOutput returns a Reporter using the given tmux runner.
func NewStatusOutput(runner displayMessenger) *Reporter {
	return NewReporter(NewStatusSink(runner))
}
