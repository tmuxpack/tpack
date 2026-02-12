package main

import (
	"fmt"
	"os"
)

// version is set via -ldflags at build time.
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		// No args = init (backward compat: `run '~/.tmux/plugins/tpm/tpm'`).
		os.Exit(runInit())
	}

	switch os.Args[1] {
	case "init":
		os.Exit(runInit())
	case "install":
		os.Exit(runInstall(os.Args[2:]))
	case "update":
		os.Exit(runUpdate(os.Args[2:]))
	case "clean":
		os.Exit(runClean(os.Args[2:]))
	case "source":
		os.Exit(runSource())
	case "tui":
		os.Exit(runTui(os.Args[2:]))
	case "version":
		fmt.Println("tpm " + version)
	default:
		fmt.Fprintf(os.Stderr, "tpm: unknown command %q\n", os.Args[1])
		fmt.Fprintln(os.Stderr, "usage: tpm [init|install|update|clean|source|tui|version]")
		os.Exit(1)
	}
}
