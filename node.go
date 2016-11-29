package zerosvc

import (
	"github.com/satori/go.uuid"
	"sync"
	"time"
	"fmt"
)

var namespace = `63082cd1-0f91-48cd-923a-f1523a26549b`

type Node struct {
	Name string
	UUID string
	TTL  int
	sync.RWMutex
	Services map[string]Service
	Transport Transport
}

func NewNode(NodeName string, NodeUUID ...string) *Node {
	var r Node
	r.Name = NodeName
	if len(NodeUUID) > 0 {
		r.UUID = NodeUUID[0]
	} else {
		ns, _ := uuid.FromString(namespace)
		uuid := uuid.NewV5(ns, NodeName)
		r.UUID = uuid.String()
	}
	r.TTL = 120
	return &r
}

func (n *Node) SetTransport(t Transport) {
	n.Transport = t
}

// Create empty event. Note that you either need to call PrepareEvent() on it to add checksum/timestamp or add it yourself

func (node *Node) NewEvent() Event {
	var ev Event
	ev.Headers = make(map[string]interface{})
	// always define required keys
	ev.Headers["node-name"] = node.Name
	ev.Headers["node-uuid"] = node.UUID
	ev.Headers["ts"] = time.Unix(0, 0)
	ev.Headers["sha256"] = ""
	return ev
}

func (node *Node) PrepareReply(ev Event) Event {
	reply := node.NewEvent()
	has := func(key string) bool { _, ok := ev.Headers[key]; return ok }
	if has("correlation-id") {
		reply.Headers["correlation-id"] = ev.Headers["correlation-id"]
	} else if has("node-id") {
		reply.Headers["correlation-id"] = ev.Headers["node-name"].(string) + "-" + time.Now().String()
	} else {
		reply.Headers["correlation-id"] = "badmsg, recv-by: " +  node.Name + "-" + time.Now().String()
	}
	return reply
}

// convenience methods

func (n *Node) SendEvent(path string, ev Event) error {
	if n.Transport == nil {
		return fmt.Errorf("please set transport via SetTransport method")
	}
	return n.Transport.SendEvent(path, ev)
}

func (n *Node) GetEventsCh(filter string) (chan Event, error) {
	ch := make(chan Event)
	if n.Transport == nil {
		return ch, fmt.Errorf("please set transport via SetTransport method")
	}
	err := n.Transport.GetEvents(filter, ch)
	return ch, err
}
func (n *Node) GetEvents(filter string, ch chan Event) (error) {
	return n.Transport.GetEvents(filter, ch)
}
