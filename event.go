package zerosvc

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Event struct {
	// Lock for ack/nacking the message (if transport supports/requires it).
	// Unused if given transport tolerates multiple acks for same message
	ackLock     *sync.Mutex
	ReplyTo     string // ReplyTo address for RPC-like usage, if underlying transport supports it
	transport   Transport
	ack         chan ack
	Redelivered bool // whether message is first try or another one
	NeedsAck    bool
	Headers     map[string]interface{}
	Body        []byte
}

func NewEvent() Event {
	var e Event
	e.ackLock = &sync.Mutex{}
	e.Headers = make(map[string]interface{})
	return e
}

// prepare event to be sent and validate it. Includes most of the housekeeping parts like generating hash of body and ts (only if they are not present)
func (ev *Event) Prepare() error {
	var err error
	if ev.Headers[`sha256`] == "" {
		body256 := sha256.Sum256(ev.Body)
		ev.Headers[`sha256`] = string(hex.EncodeToString(body256[:32]))
	}
	if ev.Headers["ts"] == time.Unix(0, 0) {
		ev.Headers["ts"] = time.Now()
	}
	return err
}

// Serialize into JSON body
func (ev *Event) Marshal(v interface{}) error {
	out, err := json.Marshal(v)
	if err == nil {
		ev.Body = out
	}
	return err
}

// Return JSON-Deserialized Body

func (ev *Event) Unmarshal(v interface{}) error {
	err := json.Unmarshal(ev.Body, v)
	return err
}

func (ev *Event) IsRPC() bool {
	return len(ev.ReplyTo) > 0
}

func (ev *Event) Reply(reply Event) error {
	if len(ev.ReplyTo) < 1 {
		return fmt.Errorf("No reply-to header in orignal event: %+v", ev)
	}
	return ev.transport.SendReply(ev.ReplyTo, reply)
}

func (ev *Event) Ack() {
	if ev.ackLock != nil {
		ev.ackLock.Lock()
		defer ev.ackLock.Unlock()
	}
	if ev.NeedsAck {
		ev.ack <- ack{ack: true}
		ev.NeedsAck = false
	}
}

// set to true to drop instead of requeueing
func (ev *Event) Nack(Drop ...bool) error {
	if ev.ackLock != nil {
		ev.ackLock.Lock()
		defer ev.ackLock.Unlock()
	}
	if ev.NeedsAck {
		if len(Drop) > 0 {
			ev.ack <- ack{nack: true, drop: Drop[0]}
		} else {
			ev.ack <- ack{nack: true}
		}
		ev.NeedsAck = false
		return nil
	} else {
		return fmt.Errorf("tried to send Nack() on event that is in auto-acknowledge mode or was acked already!")
	}
}

type ack struct {
	ack  bool
	nack bool
	drop bool
}
