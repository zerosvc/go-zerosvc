package zerosvc

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestNew(t *testing.T) {
	type cfg struct {
		Heartbeat int
	}
	trCfg := cfg{
		Heartbeat: 3,
	}
	node, err := New(Config{
		NodeName:  "test_dummy_node",
		Transport: TransportDummy("dummy://addr", trCfg),
	})
	ev := node.NewEvent()
	ev.Body = []byte("here is some cake")
	ev.Prepare()

	Convey("Create node and connect to transport", t, func() {
		So(err, ShouldBeNil)
	})
	err = node.SendEvent("test", ev)
	Convey("Send event", t, func() {
		So(err, ShouldBeNil)
	})

}
