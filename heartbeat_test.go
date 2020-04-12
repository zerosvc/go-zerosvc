package zerosvc

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	//	"fmt"
)

func TestHeartbeat(t *testing.T) {
	node := NewNode("testnode")

	node.Services["test-service"] = Service{
		Path:        "discovery",
		Description: "info event",
		Defaults:    nil,
	}
	ev := node.NewHeartbeat()

	Convey("Heartbeat fields", t, func() {
		So(ev.Headers, ShouldContainKey, "node-uuid")
		So(string(ev.Body), ShouldContainSubstring, "discovery")
		So(string(ev.Body), ShouldContainSubstring, "info event")
		So(string(ev.Body), ShouldNotContainSubstring, "1970-01-01")
	})
	Convey("Non-zero timestampt", t, func() {
		So(string(ev.Body), ShouldNotContainSubstring, "1970-01-01")
	})
}
