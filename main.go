package main

import "github.com/octobit/heimdall-cli/cmd"

// version is set at build time via -ldflags "-X main.version=<value>".
// Falls back to "dev" when built without the flag (e.g. go run .).
var version = "dev"

// commit is the short git SHA, also injected via ldflags.
var commit = ""

func main() {
	v := version
	if commit != "" {
		v += "+" + commit
	}
	cmd.SetVersion(v)
	cmd.Execute()
}
