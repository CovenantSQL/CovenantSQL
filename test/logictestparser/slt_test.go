package logictestparser

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

func TestSQLLogicTest_Parseogictestparser(t *testing.T) {
	Convey("parse.test", t, func() {
		log.SetLevel(log.DebugLevel)
		slt := new(SQLLogicTestSuite)
		ml, err := slt.Parse("./parse.test")
		So(err, ShouldBeNil)
		So(len(ml), ShouldEqual, 15)
	})
	Convey("select1.test", t, func() {
		log.SetLevel(log.DebugLevel)
		slt := new(SQLLogicTestSuite)
		ml, err := slt.Parse("./select1.test")
		So(err, ShouldBeNil)
		So(len(ml), ShouldEqual, 1031)
	})
}
