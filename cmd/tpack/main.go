package main

import "os"

// version is set via -ldflags at build time.
var version = "dev"

// binaryName is the name of the compiled Go binary.
const binaryName = "tpack"

func main() {
	os.Exit(Execute(version))
}
