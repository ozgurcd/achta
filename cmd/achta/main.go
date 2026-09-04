package main

import (
	"os"

	"github.com/ozgurcd/achta/internal/buildinfo"
	"github.com/ozgurcd/achta/internal/cli"
)

var version = buildinfo.Version

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}
