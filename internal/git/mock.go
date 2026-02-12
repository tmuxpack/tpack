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
	Calls    []FetchOptions
	FetchErr error
	Outdated map[string]bool
}

func NewMockFetcher() *MockFetcher {
	return &MockFetcher{Outdated: make(map[string]bool)}
}

func (m *MockFetcher) Fetch(_ context.Context, opts FetchOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, opts)
	return m.FetchErr
}

func (m *MockFetcher) IsOutdated(dir string) (bool, error) {
	return m.Outdated[dir], nil
}
