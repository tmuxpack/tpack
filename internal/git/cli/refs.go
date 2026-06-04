package cli

import "regexp"

var commitSHArgx = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

func looksLikeCommitSHA(ref string) bool {
	return commitSHArgx.MatchString(ref)
}
