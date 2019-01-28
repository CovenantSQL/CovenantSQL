package metric

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/utils"
)

func TestInitMetricWeb(t *testing.T) {
	Convey("init metric web", t, func() {
		ports, err := utils.GetRandomPorts("127.0.0.1", 1025, 60000, 1)
		So(err, ShouldBeNil)
		addr := fmt.Sprintf("127.0.0.1:%d", ports[0])
		err = InitMetricWeb(addr)
		So(err, ShouldBeNil)
		time.Sleep(7 * time.Second)
		resp, err := http.Get("http://" + addr + "/debug/metrics")
		So(err, ShouldBeNil)
		buf := make([]byte, 40960)
		_, err = resp.Body.Read(buf)
		So(err, ShouldBeNil)
		So(string(buf), ShouldContainSubstring, "cpu_count")
		So(string(buf), ShouldContainSubstring, "fs_avail")
		So(string(buf), ShouldContainSubstring, "go:alloc")
	})
}
