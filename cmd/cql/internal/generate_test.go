package internal

import (
	"github.com/CovenantSQL/CovenantSQL/utils"
	. "github.com/smartystreets/goconvey/convey"
	"path/filepath"
	"testing"
)

var (
	baseDir = utils.GetProjectSrcDir()
	FJ      = filepath.Join
)

func TestGenerate(t *testing.T) {
	Convey("test generate", t, func(c C) {
		//FJ(baseDir, "test/node_c")
		runGenerate(CmdGenerate, []string{""})
		//runGenerate(CmdGenerate, []string{"~/.cql"})
	})
}
