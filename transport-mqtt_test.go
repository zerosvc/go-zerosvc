package zerosvc

import (
	"crypto/rand"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
	"time"
)

func TestMQTTransport(t *testing.T) {
	c := TransportMQTTConfig{}
	node := NewNode("testnode", "77ab2b23-4f1b-4247-be45-dcc2d93ffb94")
	ev := node.NewEvent()
	ev.Body = []byte("here is some cake")
	ev.Prepare()
	// default mqtt credentials
	mqttAddr := "tcp://127.0.0.1:1883"
	if len(os.Getenv("MQTT_URL")) > 0 {
		mqttAddr = os.Getenv("MQTT_URL")
	}
	tr := NewTransport(TransportMQTT, mqttAddr, c)
	_ = c

	conn_err := tr.Connect()
	if conn_err != nil {
		SkipConvey(fmt.Sprintf("Can't connect to default mqtt on [%s]", mqttAddr), t, func() {})
		return
	}
	Convey("Connection successful", t, func() {
		So(conn_err, ShouldEqual, nil)
	})
	ch := make(chan Event, 1)
	var pathName string
	rndBytes := make([]byte, 16)
	_, err := rand.Read(rndBytes)
	if err != nil {
		fmt.Println("error:", err)
		pathName = "test"
	} else {
		pathName = fmt.Sprintf("test/%X/%X/%X/%X", rndBytes[0:4], rndBytes[4:8], rndBytes[8:12], rndBytes[12:16])
	}

	chan_err := tr.GetEvents(pathName, ch)
	Convey("Setup receive channel", t, func() {
		So(chan_err, ShouldEqual, nil)
	})
	Convey("send event", t, func() {
		send_err := tr.SendEvent(pathName, ev)
		So(send_err, ShouldEqual, nil)
	})
	Convey("receive event", t, func() {
		var recv_ev Event
		timed_out := false
		select {
		case recv_ev = <-ch:
		case <-time.After(time.Second * 3):
			timed_out = true
		}
		Convey("should receive sent event", func() {
			So(timed_out, ShouldEqual, false)
		})
		So(string(recv_ev.Body), ShouldResemble, string(ev.Body))
		So(recv_ev.Headers["node-uuid"], ShouldResemble, ev.Headers["node-uuid"])
		recv_ev.ReplyTo = "debug"
		reply := node.PrepareReply(recv_ev)
		// TODO testme
		reply.Body = []byte("this is reply")
		err := recv_ev.Reply(reply)
		So(err, ShouldEqual, nil)

	})

}
