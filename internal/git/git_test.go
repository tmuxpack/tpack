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

func TestCloneWithFallbackPrimarySucceeds(t *testing.T) {
	cloner := git.NewMockCloner()
	normalize := func(url string) string { return url + "-normalized" }

	err := git.CloneWithFallback(context.Background(), cloner, git.CloneOptions{
		URL: "https://example.com/repo.git",
		Dir: "/tmp/test",
	}, normalize)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cloner.Calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(cloner.Calls))
	}
}

func TestCloneWithFallbackFallsBack(t *testing.T) {
	callCount := 0
	cloner := &failOnceMockCloner{count: &callCount}
	normalize := func(url string) string { return url + "-normalized" }

	err := git.CloneWithFallback(context.Background(), cloner, git.CloneOptions{
		URL: "https://example.com/repo.git",
		Dir: "/tmp/test",
	}, normalize)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestCloneWithFallbackBothFail(t *testing.T) {
	cloner := git.NewMockCloner()
	cloner.Err = errors.New("clone failed")
	normalize := func(url string) string { return url + "-normalized" }

	err := git.CloneWithFallback(context.Background(), cloner, git.CloneOptions{
		URL: "https://example.com/repo.git",
		Dir: "/tmp/test",
	}, normalize)
	if err == nil {
		t.Fatal("expected error when both attempts fail")
	}
}

// failOnceMockCloner fails the first call and succeeds on subsequent calls.
type failOnceMockCloner struct {
	count *int
}

func (c *failOnceMockCloner) Clone(_ context.Context, _ git.CloneOptions) error {
	*c.count++
	if *c.count <= 1 {
		return errors.New("first attempt failed")
	}
	return nil
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

func TestCLIRevParserImplementsRevParser(t *testing.T) {
	var _ git.RevParser = (*git.CLIRevParser)(nil)
}

func TestCLILoggerImplementsLogger(t *testing.T) {
	var _ git.Logger = (*git.CLILogger)(nil)
}

func TestMockRevParserImplementsRevParser(t *testing.T) {
	var _ git.RevParser = (*git.MockRevParser)(nil)
}

func TestMockLoggerImplementsLogger(t *testing.T) {
	var _ git.Logger = (*git.MockLogger)(nil)
}

func TestMockRevParser(t *testing.T) {
	r := git.NewMockRevParser()
	r.Hash = "deadbeef"

	hash, err := r.RevParse(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "deadbeef" {
		t.Errorf("expected deadbeef, got %s", hash)
	}
	if len(r.Calls) != 1 || r.Calls[0] != "/repo" {
		t.Errorf("unexpected calls: %v", r.Calls)
	}
}

func TestMockLogger(t *testing.T) {
	l := git.NewMockLogger()
	l.Commits = []git.Commit{
		{Hash: "abc", Message: "test commit"},
	}

	commits, err := l.Log(context.Background(), "/repo", "aaa", "bbb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if commits[0].Hash != "abc" || commits[0].Message != "test commit" {
		t.Errorf("unexpected commit: %+v", commits[0])
	}
}

func TestCLIValidatorRealDir(t *testing.T) {
	v := git.NewCLIValidator()
	// /tmp is not a git repo
	if v.IsGitRepo("/tmp") {
		t.Error("/tmp should not be a git repo")
	}
}
