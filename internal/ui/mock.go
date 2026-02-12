package ui

import "sync"

// MockOutput records output calls for testing.
type MockOutput struct {
	mu       sync.Mutex
	OkMsgs   []string
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

func (m *MockOutput) HasFailed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.failed
}
