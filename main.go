package main

import (
	"os"

	"github.com/kennyg/gh-annotate/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
