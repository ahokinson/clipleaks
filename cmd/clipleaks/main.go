package main

import (
	"os"

	"github.com/ahokinson/clipleaks/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	versionInfo := cli.VersionInfo{
		Version: version,
		Commit:  commit,
	}

	app := cli.New(os.Args[1:], os.Stdout, os.Stderr, versionInfo)
	os.Exit(app.Run())
}
