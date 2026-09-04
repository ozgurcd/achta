package main

import (
	"fmt"
	"os"

	"github.com/ozgurcd/achta/internal/homebrewcask"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: homebrew-cask RELEASE_JSON GENERATED_CASK")
		os.Exit(2)
	}
	releaseJSON, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read release metadata: %v\n", err)
		os.Exit(1)
	}
	cask, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read generated cask: %v\n", err)
		os.Exit(1)
	}
	rewritten, err := homebrewcask.Rewrite(cask, releaseJSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rewrite generated cask: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(rewritten); err != nil {
		fmt.Fprintf(os.Stderr, "write rewritten cask: %v\n", err)
		os.Exit(1)
	}
}
