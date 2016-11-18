package zerosvc

import (
	"github.com/satori/go.uuid"
	"sync"
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
// convenience methods

func (n *Node) SendEvent(path string, ev Event) error {
	return n.Transport.SendEvent(path, ev)
}

func (n *Node) GetEventsCh(filter string) (chan Event, error) {
	ch := make(chan Event)
	err := n.Transport.GetEvents(filter, ch)
	return ch, err
}
func (n *Node) GetEvents(filter string, ch chan Event) (error) {
	return n.Transport.GetEvents(filter, ch)
}
