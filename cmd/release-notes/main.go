package main

import (
	"fmt"
	"os"

	"github.com/ozgurcd/achta/internal/releasenotes"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: release-notes FILE vMAJOR.MINOR.PATCH")
		os.Exit(2)
	}
	info, err := os.Lstat(os.Args[1])
	if err != nil || !info.Mode().IsRegular() || info.Size() > releasenotes.MaxFile {
		fmt.Fprintln(os.Stderr, "release-notes: input must be a bounded regular file")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "release-notes: cannot read input")
		os.Exit(2)
	}
	section, err := releasenotes.Extract(data, os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "release-notes: %v\n", err)
		os.Exit(2)
	}
	if _, err := os.Stdout.Write(section); err != nil {
		fmt.Fprintln(os.Stderr, "release-notes: cannot write output")
		os.Exit(2)
	}
}
