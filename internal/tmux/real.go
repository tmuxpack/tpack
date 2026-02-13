package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

// RealRunner executes tmux commands via os/exec.
type RealRunner struct {
	version int // parsed tmux version digits (e.g., 34 for tmux 3.4)
}

// NewRealRunner returns a new RealRunner.
func NewRealRunner() *RealRunner {
	r := &RealRunner{}
	if verStr, err := r.Version(); err == nil {
		r.version = ParseVersionDigits(verStr)
	}
	return r
}

func (r *RealRunner) runTmux(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...) //nolint:noctx // tmux calls are local, no cancellation needed
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (r *RealRunner) ShowOption(option string) (string, error) {
	return r.runTmux("show-option", "-gqv", option)
}

func (r *RealRunner) ShowEnvironment(name string) (string, error) {
	out, err := r.runTmux("start-server", ";", "show-environment", "-g", name)
	if err != nil {
		return "", err
	}
	// Output format: NAME=value
	if idx := strings.Index(out, "="); idx >= 0 {
		return out[idx+1:], nil
	}
	return "", fmt.Errorf("environment variable %s not set", name)
}

func (r *RealRunner) SetEnvironment(name, value string) error {
	_, err := r.runTmux("set-environment", "-g", name, value)
	return err
}

func (r *RealRunner) BindKey(key, cmd, description string) error {
	args := []string{"bind-key"}
	if description != "" && r.version >= 31 {
		args = append(args, "-N", description)
	}
	args = append(args, key, "run-shell", cmd)
	_, err := r.runTmux(args...)
	return err
}

func (r *RealRunner) SourceFile(path string) error {
	_, err := r.runTmux("source-file", path)
	return err
}

func (r *RealRunner) DisplayMessage(msg string) error {
	_, err := r.runTmux("display-message", msg)
	return err
}

func (r *RealRunner) RunShell(cmd string) error {
	_, err := r.runTmux("run-shell", cmd)
	return err
}

func (r *RealRunner) CommandPrompt(prompt, template string) error {
	_, err := r.runTmux("command-prompt", "-p", prompt, template)
	return err
}

func (r *RealRunner) Version() (string, error) {
	return r.runTmux("-V")
}

func (r *RealRunner) StartServer() error {
	_, err := r.runTmux("start-server")
	return err
}

func (r *RealRunner) ShowWindowOption(option string) (string, error) {
	return r.runTmux("show", "-gw", option)
}

func (r *RealRunner) SetOption(option, value string) error {
	_, err := r.runTmux("set-option", "-gq", option, value)
	return err
}
