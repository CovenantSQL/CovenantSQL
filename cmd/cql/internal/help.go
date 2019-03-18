package internal

import (
	"fmt"
	"runtime"
)

const name = "cql"

var (
	// Version of command, set by main func of version
	Version = "unknown"
)

// CmdVersion is cql version command entity.
var CmdVersion = &Command{
	UsageLine:   "cql version",
	Description: "Show cql build version infomation",
}

func init() {
	CmdVersion.Run = runVersion
}

// PrintVersion prints program git version.
func PrintVersion(printLog bool) string {
	version := fmt.Sprintf("%v %v %v %v %v\n",
		name, Version, runtime.GOOS, runtime.GOARCH, runtime.Version())

	if printLog {
		ConsoleLog.Infof("cql build: %s", version)
	}

	return version
}

func runVersion(cmd *Command, args []string) {
	fmt.Print(PrintVersion(false))
}
