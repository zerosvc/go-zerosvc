package zerosvc

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	g "github.com/XANi/goneric"
	"github.com/fxamacker/cbor/v2"
	uuid "github.com/satori/go.uuid"
	"strings"
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
	tr               Transport
	e                Encoder
	d                Decoder
	autoTrace        bool
}

type NodeInfo struct {
	Name     string
	UUID     string
	Services map[string]Service
}

func NewNode(config Config) (*Node, error) {
	n := Node{
		Name:      config.NodeName,
		UUID:      config.NodeUUID,
		Services:  map[string]Service{},
		e:         config.Encoder,
		d:         config.Decoder,
		autoTrace: true,
	}
	if len(config.NodeUUID) == 0 {
		n.UUID = uuid.NewV5(namespace, n.Name).String()
	}
	if config.Encoder == nil {
		n.e, _ = cbor.CanonicalEncOptions().EncMode()
	}
	if config.Decoder == nil {
		n.d, _ = cbor.DecOptions{}.DecMode()
	}
	if config.Transport == nil {
		return nil, fmt.Errorf("transport required in config")
	}
	n.discoveryPath = strings.Join([]string{"discovery", n.Name, n.UUID}, "/")
	n.tr = config.Transport
	err := n.tr.Connect(Hooks{}, n.eventRoot+"/"+n.discoveryPath)
	if err == nil {
		n.Heartbeat()
	}
	return &n, err
}

func (n *Node) NewEvent(traceSpanId ...[]byte) Event {
	ev := Event{
		NodeName:  n.Name,
		NodeUUID:  n.UUID,
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
	data, err := ev.Serialize()
	if err != nil {
		return err
	}
	return n.tr.Publish(
		Message{
			Topic:           n.eventRoot + "/" + path,
			ResponseTopic:   "",
			CorrelationData: nil,
			ContentType:     "",
			Metadata:        nil,
			Payload:         data,
			Retain:          ev.retain,
		},
	)
}

func (n *Node) Heartbeat() {
	ev := n.NewEvent()
	v := NodeInfo{
		Name:     n.Name,
		UUID:     n.UUID,
		Services: n.Services,
	}
	d, _ := json.Marshal(v)
	ev.Body = d
	ev.retain = true
	n.SendEvent(n.discoveryPath, ev)
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
