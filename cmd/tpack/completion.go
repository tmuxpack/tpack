package main

import "github.com/spf13/cobra"

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate shell completion scripts",
	Long: `Generate completion scripts for bash, zsh, or fish.

To load completions:

Bash:
  $ source <(tpack completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ tpack completion bash > /etc/bash_completion.d/tpack
  # macOS:
  $ tpack completion bash > $(brew --prefix)/etc/bash_completion.d/tpack

Zsh:
  $ tpack completion zsh > "${fpath[1]}/_tpack"

  # You may need to restart your shell or run:
  $ compinit

Fish:
  $ tpack completion fish | source

  # To load completions for each session, execute once:
  $ tpack completion fish > ~/.config/fish/completions/tpack.fish
`,
}

var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion script",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenBashCompletionV2(cmd.OutOrStdout(), true)
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completion script",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(cmd.OutOrStdout())
	},
}

var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completion script",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
	},
}

func init() {
	completionCmd.AddCommand(completionBashCmd, completionZshCmd, completionFishCmd)
}
