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

var CmdVersion = &Command{
	UsageLine:   "cql version",
	Description: "Show cql build version infomation",
}

func init() {
	CmdVersion.Run = runVersion
}

func runVersion(cmd *Command, args []string) {
	fmt.Printf("%v %v %v %v %v\n",
		name, Version, runtime.GOOS, runtime.GOARCH, runtime.Version())
}
