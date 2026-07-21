package ui

// Level identifies a message's severity.
type Level uint8

const (
	LevelInfo Level = iota
	LevelWarning
	LevelError
)

// Message is a message delivered to a Sink.
type Message struct {
	Level Level
	Text  string
}

// Sink delivers output messages to a transport.
type Sink interface {
	Write(Message) error
	EndMessage() error
}
