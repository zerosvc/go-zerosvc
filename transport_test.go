package zerosvc

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestDummyTransport(t *testing.T) {
	type cfg struct {
		Heartbeat int
	}
	c := cfg{
		Heartbeat: 3,
	}
	node := NewNode("testnode", "77ab2b23-4f1b-4247-be45-dcc2d93ffb94")
	ev := node.NewEvent()
	ev.Body = []byte("here is some cake")
	ev.Prepare()

	tr := NewTransport(TransportDummy, "dummy:addr", c)

	conn_err := tr.Connect()
	Convey("Connection successful", t, func() {
		So(conn_err, ShouldEqual, nil)
	})

	Convey("send event", t, func() {
		tr.SendEvent("test", ev)
	})
	ch := make(chan Event, 2)
	chan_err := tr.GetEvents("dummy", ch)
	Convey("get event", t, func() {
		So(chan_err, ShouldEqual, nil)
		recv_ev := <-ch
		So(recv_ev.Headers["node-uuid"], ShouldResemble, "77ab2b23-4f1b-4247-be45-dcc2d93ffb94")
	})
}
