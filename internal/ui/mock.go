package ui

import "sync"

// MockOutput records output calls for testing.
type MockOutput struct {
	mu       sync.Mutex
	OkMsgs   []string
	WarnMsgs []string
	ErrMsgs  []string
	EndCalls int
	failed   bool
}

// NewMockOutput returns a new MockOutput.
func NewMockOutput() *MockOutput {
	return &MockOutput{}
}

func (m *MockOutput) Ok(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OkMsgs = append(m.OkMsgs, msg)
}

func (m *MockOutput) Warn(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WarnMsgs = append(m.WarnMsgs, msg)
}

func (m *MockOutput) Err(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failed = true
	m.ErrMsgs = append(m.ErrMsgs, msg)
}

func (m *MockOutput) EndMessage() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EndCalls++
}

// Result returns ErrReported after an error message has been recorded.
func (m *MockOutput) Result() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failed {
		return ErrReported
	}
	return nil
}

// MockSink records sink calls for testing.
type MockSink struct {
	mu       sync.Mutex
	Messages []Message
	EndCalls int
	Err      error
}

// NewMockSink returns a new MockSink.
func NewMockSink() *MockSink { return &MockSink{} }

// Write records message and returns the configured error.
func (m *MockSink) Write(message Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, message)
	return m.Err
}

// EndMessage records the call and returns the configured error.
func (m *MockSink) EndMessage() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EndCalls++
	return m.Err
}

// Texts returns a snapshot of recorded message texts.
func (m *MockSink) Texts() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	texts := make([]string, 0, len(m.Messages))
	for _, message := range m.Messages {
		texts = append(texts, message.Text)
	}
	return texts
}
