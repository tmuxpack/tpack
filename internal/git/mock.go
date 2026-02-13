package git

import (
	"context"
	"sync"
)

// MockCloner records clone calls for testing.
type MockCloner struct {
	mu    sync.Mutex
	Calls []CloneOptions
	Err   error
}

func NewMockCloner() *MockCloner {
	return &MockCloner{}
}

func (m *MockCloner) Clone(_ context.Context, opts CloneOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, opts)
	return m.Err
}

// MockPuller records pull calls for testing.
type MockPuller struct {
	mu     sync.Mutex
	Calls  []PullOptions
	Output string
	Err    error
}

func NewMockPuller() *MockPuller {
	return &MockPuller{}
}

func (m *MockPuller) Pull(_ context.Context, opts PullOptions) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, opts)
	return m.Output, m.Err
}

// MockValidator returns configurable results for testing.
type MockValidator struct {
	Valid map[string]bool
}

func NewMockValidator() *MockValidator {
	return &MockValidator{Valid: make(map[string]bool)}
}

func (m *MockValidator) IsGitRepo(dir string) bool {
	return m.Valid[dir]
}

// MockFetcher returns configurable results for testing.
type MockFetcher struct {
	mu       sync.Mutex
	Calls    []string
	Err      error
	Outdated map[string]bool
}

func NewMockFetcher() *MockFetcher {
	return &MockFetcher{Outdated: make(map[string]bool)}
}

func (m *MockFetcher) IsOutdated(_ context.Context, dir string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, dir)
	if m.Err != nil {
		return false, m.Err
	}
	return m.Outdated[dir], nil
}

// MockRevParser returns configurable results for testing.
type MockRevParser struct {
	mu    sync.Mutex
	Calls []string
	Hash  string
	Err   error
}

func NewMockRevParser() *MockRevParser {
	return &MockRevParser{Hash: "abc123"}
}

func (m *MockRevParser) RevParse(_ context.Context, dir string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, dir)
	return m.Hash, m.Err
}

// MockLogger returns configurable results for testing.
type MockLogger struct {
	mu      sync.Mutex
	Calls   []mockLogCall
	Commits []Commit
	Err     error
}

type mockLogCall struct {
	Dir     string
	FromRef string
	ToRef   string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

func (m *MockLogger) Log(_ context.Context, dir, fromRef, toRef string) ([]Commit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, mockLogCall{Dir: dir, FromRef: fromRef, ToRef: toRef})
	return m.Commits, m.Err
}
