package zerosvc

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/satori/go.uuid"
	"log"
	"strings"
	"sync"
	"time"
)

var namespace = `63082cd1-0f91-48cd-923a-f1523a26549b`


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
	heartbeatEnabled bool

	Transport Transport
}

func NewNode(NodeName string, NodeUUID ...string) *Node {
	var r Node
	r.Name = NodeName
	r.Services = make(map[string]Service, 0)
	r.Info = make(map[string]interface{}, 0)
	if len(NodeUUID) > 0 {
		r.UUID = NodeUUID[0]
	} else {
		ns, _ := uuid.FromString(namespace)
		uuid := uuid.NewV5(ns, NodeName)
		r.UUID = uuid.String()
	}
	r.TTL = 120 * time.Second
	return &r
}


func (n *Node) SetTransport(t Transport) *Node {
	n.Transport = t
	return n
}

func (n *Node) HeartbeatPath() string {
	return "discovery/node-" + n.Name
}

// Create empty event. Note that you either need to call PrepareEvent() on it to add checksum/timestamp or add it yourself

func (n *Node) NewEvent() Event {
	var ev Event
	ev.Headers = make(map[string]interface{})
	// always define required keys
	ev.Headers["node-name"] = n.Name
	ev.Headers["node-uuid"] = n.UUID
	ev.Headers["ts"] = time.Unix(0, 0)
	ev.Headers["sha256"] = ""
	ev.transport = n.Transport
	return ev
}

func (n *Node) PrepareReply(ev Event) Event {
	reply := n.NewEvent()
	has := func(key string) bool { _, ok := ev.Headers[key]; return ok }
	if has("correlation-id") {
		reply.Headers["correlation-id"] = ev.Headers["correlation-id"]
	} else if has("n-id") {
		reply.Headers["correlation-id"] = ev.Headers["n-name"].(string) + "-" + time.Now().String()
	} else {
		reply.Headers["correlation-id"] = "badmsg, recv-by: " + n.Name + "-" + time.Now().String()
	}
	return reply
}

func (n *Node) SignEvent(ev *Event) error {
	if n.Signer == nil {
		return fmt.Errorf("tried to sign without defined signer")
	}
	rawSig := n.Signer.Sign(ev.Body)

	sigPkt := []byte{n.Signer.Type(), uint8(len(rawSig))}
	sigPkt = append(sigPkt, rawSig...)
	ev.Headers["_sec-sig"] = base64.StdEncoding.EncodeToString(sigPkt)
	return nil
}

func (n *Node) VerifyEvent(ev *Event) (ok bool, err error) {
	if n.PubkeyRetriever == nil {
		return false, fmt.Errorf("tried to verify without a way to retrieve signatures")
	}
	if v, ok := ev.Headers["_sec-sig"].(string); ok {
		sig, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return false, err
		}
		if len(sig) < 4 {
			return false, fmt.Errorf("sig too short")
		}
		t := uint8(sig[0])
		l := uint8(sig[1])
		vSig := sig[2:]
		if int(l) != len(vSig) {
			return false, fmt.Errorf("sig length mismatch: %d/%d", l+1, len(vSig))
		}
		switch t {
		case 0:
			return false, fmt.Errorf("sig invalid, wrong type")
		case SigTypeEd25519, SigTypeX509:
			verifier, found := n.PubkeyRetriever(n.Name, n.UUID)
			if !found {
				return false, nil
			} else if verifier == nil {
				return false, fmt.Errorf("logic returned verifier empty but found!")
			} else {
				return verifier.Verify(ev.Body, vSig), nil
			}
		default:
			return false, fmt.Errorf("sig type ID %d unsupported", t)
		}
	} else {
		return false, fmt.Errorf("expected _ev-sig string header")
	}
}

// convenience methods

func (n *Node) SendEvent(path string, ev Event) error {
	if n.Transport == nil {
		return fmt.Errorf("please set transport via SetTransport method")
	}
	if n.Signer != nil {
		err := n.SignEvent(&ev)
		if err != nil {
			return err
		}
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
func (n *Node) GetEvents(filter string, ch chan Event) error {
	return n.Transport.GetEvents(filter, ch)
}

// GetReplyChan() returns randomly generated channel for replies
func (n *Node) GetReplyChan() (path string, replyCh chan Event, err error) {
	id := mapBytesToTopicTitle(rngBlob(16))
	path = "reply/node-" + n.Name + "/" + id
	rspCh, err := n.GetEventsCh(path + "/#")
	return path, rspCh, err
}

func rngBlob(bytes int) []byte {
	rnd := make([]byte, bytes)
	i, err := rand.Read(rnd)
	if err == nil && i == bytes {
		return rnd
	}
	var errctr uint8
	var readctr = i
	for {
		errctr++
		if errctr > 10 {
			log.Panicf("could not get data from RNG")
		}
		i, err := rand.Read(rnd[readctr:])
		if i > 0 {
			readctr += i
		} else {
			panic(fmt.Sprintf("error getting RNG: %s", err))
		}
		if readctr >= bytes {
			return rnd
		}
	}
}
var base64Replacer = strings.NewReplacer(
	"+", "",
	"/", "",
	"=","",
)


// MapBytesToTopicTitle maps binary data to topic-friendly subset of characters.
func mapBytesToTopicTitle(data []byte) string {
	str := base64.StdEncoding.EncodeToString(data)
	return base64Replacer.Replace(str)
}
