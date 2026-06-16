package main

import (
	"runtime/debug"

	"github.com/effectivai/riskos-cli/cmd"
)

// set via -ldflags "-X main.version=..." at build time
var version = "dev"

func main() {
	cmd.SetVersion(resolveVersion())
	cmd.Execute()
}

func resolveVersion() string {
	//goland:noinspection GoBoolExpressions
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}
