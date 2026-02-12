package git_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tmux-plugins/tpm/internal/git"
)

func TestMockClonerImplementsCloner(t *testing.T) {
	var _ git.Cloner = (*git.MockCloner)(nil)
}

func TestMockPullerImplementsPuller(t *testing.T) {
	var _ git.Puller = (*git.MockPuller)(nil)
}

func TestMockValidatorImplementsValidator(t *testing.T) {
	var _ git.Validator = (*git.MockValidator)(nil)
}

func TestCLIClonerImplementsCloner(t *testing.T) {
	var _ git.Cloner = (*git.CLICloner)(nil)
}

func TestCLIPullerImplementsPuller(t *testing.T) {
	var _ git.Puller = (*git.CLIPuller)(nil)
}

func TestCLIValidatorImplementsValidator(t *testing.T) {
	var _ git.Validator = (*git.CLIValidator)(nil)
}

func TestFallbackClonerPrimarySucceeds(t *testing.T) {
	primary := git.NewMockCloner()
	secondary := git.NewMockCloner()
	fallback := git.NewFallbackCloner(primary, secondary)

	err := fallback.Clone(context.Background(), git.CloneOptions{
		URL: "https://example.com/repo.git",
		Dir: "/tmp/test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(primary.Calls) != 1 {
		t.Errorf("expected 1 primary call, got %d", len(primary.Calls))
	}
	if len(secondary.Calls) != 0 {
		t.Errorf("expected 0 secondary calls, got %d", len(secondary.Calls))
	}
}

func TestFallbackClonerFallsBack(t *testing.T) {
	primary := git.NewMockCloner()
	primary.Err = errors.New("primary failed")
	secondary := git.NewMockCloner()
	fallback := git.NewFallbackCloner(primary, secondary)

	err := fallback.Clone(context.Background(), git.CloneOptions{
		URL: "https://example.com/repo.git",
		Dir: "/tmp/test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(primary.Calls) != 1 {
		t.Errorf("expected 1 primary call, got %d", len(primary.Calls))
	}
	if len(secondary.Calls) != 1 {
		t.Errorf("expected 1 secondary call, got %d", len(secondary.Calls))
	}
}

func TestFallbackClonerBothFail(t *testing.T) {
	primary := git.NewMockCloner()
	primary.Err = errors.New("primary failed")
	secondary := git.NewMockCloner()
	secondary.Err = errors.New("secondary failed")
	fallback := git.NewFallbackCloner(primary, secondary)

	err := fallback.Clone(context.Background(), git.CloneOptions{
		URL: "https://example.com/repo.git",
		Dir: "/tmp/test",
	})
	if err == nil {
		t.Fatal("expected error when both fail")
	}
}

func TestMockValidator(t *testing.T) {
	v := git.NewMockValidator()
	v.Valid["/repo"] = true

	if !v.IsGitRepo("/repo") {
		t.Error("expected /repo to be valid")
	}
	if v.IsGitRepo("/not-repo") {
		t.Error("expected /not-repo to be invalid")
	}
}

func TestCLIValidatorRealDir(t *testing.T) {
	v := git.NewCLIValidator()
	// /tmp is not a git repo
	if v.IsGitRepo("/tmp") {
		t.Error("/tmp should not be a git repo")
	}
}
