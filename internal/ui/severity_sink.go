package ui

type severitySink struct {
	info    Sink
	warning Sink
	failure Sink
}

// NewSeveritySink returns a sink that routes messages according to severity.
func NewSeveritySink(info, warning, failure Sink) Sink {
	return &severitySink{info: info, warning: warning, failure: failure}
}

func (s *severitySink) Write(message Message) error {
	switch message.Level {
	case LevelWarning:
		return s.warning.Write(message)
	case LevelError:
		return s.failure.Write(message)
	case LevelInfo:
		return s.info.Write(message)
	default:
		return s.info.Write(message)
	}
}

func (s *severitySink) EndMessage() error { return s.info.EndMessage() }
