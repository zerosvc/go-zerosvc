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
	senderNode := NewNode("testnode3", "f620768c-286c-4f41-9084-3cbd957235fe")
	// default rabbitmq credentials
	amqpAddr := "amqp://guest:guest@localhost:5672"
	if len(os.Getenv("AMQP_URL")) > 0 {
		amqpAddr = os.Getenv("AMQP_URL")
	}

	senderTr := NewTransport(TransportAMQP, amqpAddr, c)
	err := senderTr.Connect()
	require.Nil(t, err)

	senderTr.AdminCleanup()
	senderNode.SetTransport(senderTr)

	readers := 5

	msgs := make([][]Event, readers)
	end := make(chan bool, readers)
	wg := sync.WaitGroup{}
	readersCount := make([]int, readers)
	for i := 0; i < readers; i++ {
		msgs[i] = make([]Event, 0)
		name := fmt.Sprintf("test-node-%d", i)
		node := NewNode(name)
		tr := NewTransport(TransportAMQP, amqpAddr, c)
		err := tr.Connect()
		require.Nil(t, err)
		node.SetTransport(tr)
		ch := make(chan Event, 1)
		ch_err := tr.GetEvents("service.test.send.#", ch)
		require.Nil(t, ch_err)

		wg.Add(1)
		go func(readerId int) {
			for ev := range ch {
				readersCount[readerId]++
				time.Sleep(time.Millisecond * 10 * time.Duration(readerId+1))
				ev.Ack()
				if string(ev.Body) == "end" {
					wg.Done()
					end <- true
					return
				} else {
					msgs[readerId] = append(msgs[readerId], ev)
				}
			}
			wg.Done()
			return
		}(i)

	}
	for i := 0; i < 100; i++ {
		sendEnd := senderNode.NewEvent()
		sendEnd.Body = []byte(fmt.Sprintf("m:%d", i))
		sendEnd.Prepare()
		err = senderNode.SendEvent("service.test.send", sendEnd)
	}
	for i := 0; i < readers; i++ {
		// send end twice so both threads go away
		sendEnd := senderNode.NewEvent()
		sendEnd.Body = []byte("end")
		sendEnd.Prepare()
		err = senderNode.SendEvent("service.test.send", sendEnd)
		require.Nil(t, err)
	}
	select {
	case <-end:
		wgEnd := make(chan bool, 1)
		go func() {
			wg.Wait()
			wgEnd <- true
		}()
		select {
		case <-wgEnd:
		case <-time.After(time.Second * 6):
			t.Fatalf("timed out waiting for reader threads, received messages: [%+v]", readersCount)
		}
	case <-time.After(time.Second * 6):
		list := make([]string, 0)
		for _, m := range msgs {
			for _, v := range m {
				list = append(list, string(v.Body))
			}
		}
		t.Fatalf("no end message received from MQ, got: %+v", list)

	}
	msgCount := 0
	for _, m := range msgs {
		for _, _ = range m {
			msgCount++
		}
	}

	assert.Equal(t, 100, msgCount, "should get 100 messages")

}
