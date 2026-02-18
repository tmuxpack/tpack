package main

import (
	"fmt"
	"os"
)

// version is set via -ldflags at build time.
var version = "dev"

// binaryName is the name of the compiled Go binary.
const binaryName = "tpack"

func main() {
	if len(os.Args) < 2 {
		// No args = init (backward compat: `run '~/.tmux/plugins/tpm/tpm'`).
		os.Exit(runInit())
	}

	switch os.Args[1] {
	case "init":
		os.Exit(runInit())
	case "install": // TODO: add --all flag
		os.Exit(runInstall(os.Args[2:]))
	case "update":
		os.Exit(runUpdate(os.Args[2:]))
	case "clean":
		os.Exit(runClean(os.Args[2:]))
	case "source":
		os.Exit(runSource())
	case "tui":
		os.Exit(runTui(os.Args[2:]))
	case "commits":
		os.Exit(runCommits(os.Args[2:]))
	case "check-updates":
		os.Exit(runCheckUpdates())
	case "self-update":
		os.Exit(runSelfUpdate())
	case "version":
		fmt.Println("tpack " + version)
	default:
		fmt.Fprintf(os.Stderr, "tpack: unknown command %q\n", os.Args[1]) //nolint:gosec // CLI output, not web context
		fmt.Fprintln(os.Stderr, "usage: tpack [init|install|update|clean|source|tui|commits|check-updates|self-update|version]")
		os.Exit(1)
	}
}
