package tmux

import "sync"

// MockRunner records tmux calls for testing.
type MockRunner struct {
	mu sync.Mutex

	// Return values keyed by method+args.
	Options     map[string]string
	Environment map[string]string
	WindowOpts  map[string]string
	VersionStr  string
	Errors      map[string]error

	// Recorded calls.
	Calls []Call
}

// Call records a single tmux method invocation.
type Call struct {
	Method string
	Args   []string
}

// NewMockRunner returns a MockRunner with initialized maps.
func NewMockRunner() *MockRunner {
	return &MockRunner{
		Options:     make(map[string]string),
		Environment: make(map[string]string),
		WindowOpts:  make(map[string]string),
		Errors:      make(map[string]error),
	}
}

func (m *MockRunner) record(method string, args ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, Call{Method: method, Args: args})
}

func (m *MockRunner) err(key string) error {
	if e, ok := m.Errors[key]; ok {
		return e
	}
	return nil
}

func (m *MockRunner) ShowOption(option string) (string, error) {
	m.record("ShowOption", option)
	return m.Options[option], m.err("ShowOption:" + option)
}

func (m *MockRunner) ShowEnvironment(name string) (string, error) {
	m.record("ShowEnvironment", name)
	val, ok := m.Environment[name]
	if !ok {
		return "", m.err("ShowEnvironment:" + name)
	}
	return val, m.err("ShowEnvironment:" + name)
}

func (m *MockRunner) SetEnvironment(name, value string) error {
	m.record("SetEnvironment", name, value)
	m.Environment[name] = value
	return m.err("SetEnvironment:" + name)
}

func (m *MockRunner) BindKey(key, cmd string) error {
	m.record("BindKey", key, cmd)
	return m.err("BindKey:" + key)
}

func (m *MockRunner) SourceFile(path string) error {
	m.record("SourceFile", path)
	return m.err("SourceFile:" + path)
}

func (m *MockRunner) DisplayMessage(msg string) error {
	m.record("DisplayMessage", msg)
	return m.err("DisplayMessage")
}

func (m *MockRunner) RunShell(cmd string) error {
	m.record("RunShell", cmd)
	return m.err("RunShell")
}

func (m *MockRunner) CommandPrompt(prompt, template string) error {
	m.record("CommandPrompt", prompt, template)
	return m.err("CommandPrompt")
}

func (m *MockRunner) Version() (string, error) {
	m.record("Version")
	return m.VersionStr, m.err("Version")
}

func (m *MockRunner) StartServer() error {
	m.record("StartServer")
	return m.err("StartServer")
}

func (m *MockRunner) ShowWindowOption(option string) (string, error) {
	m.record("ShowWindowOption", option)
	return m.WindowOpts[option], m.err("ShowWindowOption:" + option)
}

func (m *MockRunner) SetOption(option, value string) error {
	m.record("SetOption", option, value)
	m.Options[option] = value
	return m.err("SetOption:" + option)
}
