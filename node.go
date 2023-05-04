package zerosvc

import (
	"crypto/rand"
	g "github.com/XANi/goneric"
	"sync"
	"time"
)

type Node struct {
	Name string
	UUID string
	Info map[string]interface{}
	TTL  time.Duration
	sync.RWMutex
	Services map[string]Service
	// signer governs storing public/private key and decoding the signatures
	Signer           Signer
	PubkeyRetriever  func(nodeName string, nodeUUID string) (v Verifier, found bool)
	discoveryPath    string
	eventRoot        string
	heartbeatEnabled bool
	Transport        Transport
	e                Encoder
	d                Decoder
	autoTrace        bool
}

func (n *Node) NewEvent(traceSpanId ...[]byte) Event {
	ev := Event{
		NodeUUID:  n.Name,
		NodeName:  n.UUID,
		ReplyTo:   "",
		Headers:   map[string]any{},
		Signature: nil,
		Body:      nil,
		n:         n,
	}
	if len(traceSpanId) > 0 {
		ev.TraceID = traceSpanId[0]
		if len(traceSpanId) > 1 {
			ev.SpanID = traceSpanId[1]
		} else {
			ev.SpanID = make([]byte, 8)
			_, err := rand.Read(ev.SpanID)
			if err != nil {
				panic(err)
			}
		}
	} else if n.autoTrace {
		ev.TraceID = make([]byte, 16)
		g.Must(rand.Read(ev.TraceID))
		ev.TraceID = make([]byte, 8)
		g.Must(rand.Read(ev.SpanID))
	}
	return ev
}
func (n *Node) PrepareReply(ev Event) Event {
	reply := Event{
		TraceID:   ev.TraceID,
		NodeUUID:  n.UUID,
		NodeName:  n.Name,
		ReplyTo:   ev.ReplyTo,
		Headers:   map[string]any{},
		Signature: nil,
		Body:      nil,
		n:         n,
	}
	if len(reply.TraceID) > 0 {
		reply.SpanID = make([]byte, 8)
		g.Must(rand.Read(reply.SpanID))

	}
	return reply
}

func (n *Node) SendEvent(path string, ev Event) error {
	return nil
}

func (n *Node) GetEventsCh(filter string) (chan Event, error) {
	ch := make(chan Event, 1)
	return ch, nil
}

// GetReplyChan() returns randomly generated channel for replies
func (n *Node) GetReplyChan() (path string, replyCh chan Event, err error) {
	path = n.eventRoot + "/reply/" + n.Name + "/" + mapBytesToTopicTitle(rngBlob(8))
	return
}
