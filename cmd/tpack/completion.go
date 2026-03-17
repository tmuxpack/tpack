package main

import "github.com/spf13/cobra"

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate shell completion scripts",
}
