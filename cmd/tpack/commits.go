package main

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	gitcli "github.com/tmuxpack/tpack/internal/git/cli"
	"github.com/tmuxpack/tpack/internal/tui"
	"github.com/tmuxpack/tpack/internal/ui"
)

var commitsCmd = &cobra.Command{
	Use:   "commits",
	Short: "Show commit history for a plugin",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("dir")
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		name, _ := cmd.Flags().GetString("name")

		// Run git log to get commits.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		out := ui.NewShellOutput()

		logger := gitcli.NewLogger()
		commits, err := logger.Log(ctx, dir, from, to)
		if err != nil {
			out.Err("git log failed: " + err.Error())
			return errSilent
		}

		if len(commits) == 0 {
			return nil
		}

		if err := tui.RunCommitViewer(name, commits, tui.DefaultTheme()); err != nil {
			out.Err(err.Error())
			return errSilent
		}
		return nil
	},
}

func init() {
	commitsCmd.Flags().String("dir", "", "plugin directory")
	commitsCmd.Flags().String("from", "", "start commit")
	commitsCmd.Flags().String("to", "", "end commit")
	commitsCmd.Flags().String("name", "", "plugin name")

	_ = commitsCmd.MarkFlagRequired("dir")
	_ = commitsCmd.MarkFlagRequired("from")
	_ = commitsCmd.MarkFlagRequired("to")
	_ = commitsCmd.MarkFlagRequired("name")

	_ = commitsCmd.RegisterFlagCompletionFunc("name", completePluginNames)
}
