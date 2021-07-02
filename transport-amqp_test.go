package zerosvc

import (
	"crypto/rand"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"sync"
	"testing"
	"time"
)

func TestAMQPTransport(t *testing.T) {
	c := TransportAMQPConfig{
		Heartbeat:     3,
		EventExchange: "test-events",
	}
	node := NewNode("testnode", "77ab2b23-4f1b-4247-be45-dcc2d93ffb94")
	ev := node.NewEvent()
	ev.Body = []byte("here is some cake")
	ev.Prepare()
	// default rabbitmq credentials
	amqpAddr := "amqp://guest:guest@localhost:5672"
	if len(os.Getenv("AMQP_URL")) > 0 {
		amqpAddr = os.Getenv("AMQP_URL")
	}

	tr := NewTransport(TransportAMQP, amqpAddr, c)
	_ = c

	conn_err := tr.Connect()
	if conn_err != nil {
		SkipConvey(fmt.Sprintf("Can't connect to default rabbitmq on [%s]", amqpAddr), t, func() {})
		return
	}
	Convey("Connection successful", t, func() {
		So(conn_err, ShouldEqual, nil)
	})
	tr.AdminCleanup()
	ch := make(chan Event, 1)
	var pathName string
	rndBytes := make([]byte, 16)
	_, err := rand.Read(rndBytes)
	if err != nil {
		fmt.Println("error:", err)
		pathName = "test"
	} else {
		pathName = fmt.Sprintf("%X", rndBytes)
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

func TestAMQPTransport_MultiACKqueue(t *testing.T) {
	c := TransportAMQPConfig{
		Heartbeat:       3,
		NoAutoAck:       true,
		SharedQueue:     true,
		MaxInFlightRecv: 4,
		EventExchange:   "test-events",
	}
	node1 := NewNode("testnode1", "77ab2b23-4f1b-4247-be45-dcc2d93ffb94")
	node2 := NewNode("testnode2", "34e20558-7987-49d2-bc51-8f81f2c953f0")
	senderNode := NewNode("testnode3", "f620768c-286c-4f41-9084-3cbd957235fe")
	// default rabbitmq credentials
	amqpAddr := "amqp://guest:guest@localhost:5672"
	if len(os.Getenv("AMQP_URL")) > 0 {
		amqpAddr = os.Getenv("AMQP_URL")
	}

	tr1 := NewTransport(TransportAMQP, amqpAddr, c)
	err := tr1.Connect()
	require.Nil(t, err)
	tr1.AdminCleanup()

	tr2 := NewTransport(TransportAMQP, amqpAddr, c)
	err = tr2.Connect()
	require.Nil(t, err)

	senderTr := NewTransport(TransportAMQP, amqpAddr, c)
	err = senderTr.Connect()
	require.Nil(t, err)

	node1.SetTransport(tr1)
	node2.SetTransport(tr2)
	senderNode.SetTransport(senderTr)
	ch1 := make(chan Event, 1)
	ch1_err := tr1.GetEvents("service.test.send.#", ch1)
	require.Nil(t, ch1_err)
	ch2 := make(chan Event, 1)
	ch2_err := tr2.GetEvents("service.test.send.#", ch2)
	require.Nil(t, ch2_err)
	sender := make(chan Event, 1)
	sender_err := tr2.GetEvents("service.test.send.#", sender)
	require.Nil(t, sender_err)
	msgs1 := make([]Event, 0)
	msgs2 := make([]Event, 0)
	ch1end := make(chan bool, 2)
	ch2end := make(chan bool, 2)
	end := make(chan bool, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for ev := range ch1 {
			ev.Ack()
			if string(ev.Body) == "end" {
				end <- true
				wg.Done()
				return
			} else {
				msgs1 = append(msgs1, ev)

			}
		}
	}()
	go func() {
		for ev := range ch2 {
			ev.Ack()
			if string(ev.Body) == "end" {
				end <- true
				wg.Done()
				return
			} else {
				msgs1 = append(msgs1, ev)

			}
		}
		wg.Done()
	}()
	for i := 0; i < 100; i++ {
		sendEnd := senderNode.NewEvent()
		sendEnd.Body = []byte(fmt.Sprintf("m:%d", i))
		sendEnd.Prepare()
		err = node1.SendEvent("service.test.send", sendEnd)
	}

	// send end twice so both threads go away
	sendEnd := senderNode.NewEvent()
	sendEnd.Body = []byte("end")
	sendEnd.Prepare()
	err = node1.SendEvent("service.test.send", sendEnd)

	sendEnd = senderNode.NewEvent()
	sendEnd.Body = []byte("end")
	sendEnd.Prepare()
	err = node1.SendEvent("service.test.send", sendEnd)

	select {
	case <-end:
		ch1end <- true
		ch2end <- true
		wg.Wait()
	case <-time.After(time.Second * 6):
		list := make([]string, 0)
		for _, v := range msgs1 {
			list = append(list, string(v.Body))
		}
		for _, v := range msgs1 {
			list = append(list, string(v.Body))
		}
		t.Fatalf("no end message received from MQ, got: %+v", list)

	}

	assert.Equal(t, 100, len(msgs1)+len(msgs2), "should get 100 messages")

}
