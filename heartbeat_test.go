package zerosvc

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	//	"fmt"
)

func TestHeartbeat(t *testing.T) {
	node := NewNode("testnode")

	ev := node.NewHeartbeat()

	Convey("Heartbeat fields", t, func() {
		So(ev.Headers, ShouldContainKey, "node-uuid")
		So(string(ev.Body), ShouldContainSubstring, "body")
		So(string(ev.Body), ShouldNotContainSubstring, "1970-01-01")
	})
	Convey("Non-zero timestampt", t, func() {
		So(string(ev.Body), ShouldNotContainSubstring, "1970-01-01")
	})
}
