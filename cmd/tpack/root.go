package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// errSilent signals that the command has already printed errors to stderr.
// Execute() will not print it again, but will still return exit code 1.
var errSilent = errors.New("")

var rootCmd = &cobra.Command{
	Use:   "tpack",
	Short: "A modern tmux plugin manager",
	Long:  "tpack is a drop-in replacement for TPM (Tmux Plugin Manager).",

	// Don't print usage on runtime errors.
	SilenceUsage: true,
	// We handle error printing ourselves to preserve current output format.
	SilenceErrors: true,

	// Default command (no args): run init for backward compatibility.
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInitCmd()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print tpack version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("tpack " + version)
	},
}

func init() {
	rootCmd.SetVersionTemplate("tpack {{.Version}}\n")

	rootCmd.AddCommand(
		initCmd,
		installCmd,
		updateCmd,
		cleanCmd,
		sourceCmd,
		tuiCmd,
		commitsCmd,
		checkUpdatesCmd,
		selfUpdateCmd,
		completionCmd,
		versionCmd,
	)
}

// Execute runs the root command with the given version.
func Execute(v string) int {
	rootCmd.Version = v
	if err := rootCmd.Execute(); err != nil {
		if !errors.Is(err, errSilent) {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		return 1
	}
	return 0
}
