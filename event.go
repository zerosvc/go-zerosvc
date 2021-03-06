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
	ReplyTo     string `json:"reply_to,omitempty"` // ReplyTo address for RPC-like usage, if underlying transport supports it
	transport   Transport
	ack         chan ack
	Redelivered bool `json:"redelivered,omitempty"` // whether message is first try or another one
	NeedsAck    bool `json:"needs_ack,omitempty"`
	// for how long message should be retained if transport supports it
	// actual TTL will be calculated from current time difference
	RetainTill *time.Time `json:"retain_till,omitempty"`
	// incoming routing key
	RoutingKey string                 `json:"routing_key,omitempty"`
	Headers    map[string]interface{} `json:"headers"`
	Body       []byte                 `json:"body"`
	// signature of the body
	Signature string `json:"sig,omitempty"`
	prepared  bool
}

func NewEvent() Event {
	var e Event
	e.ackLock = &sync.Mutex{}
	e.Headers = make(map[string]interface{})
	return e
}

// NodeName() returns name of node that sent the event
func (ev *Event) NodeName() string {
	if val, ok := ev.Headers["node-name"].(string); ok {
		return val
	}
	return ""
}

// NodeUUID() returns UUID of node that sent the event
func (ev *Event) NodeUUID() string {
	if val, ok := ev.Headers["node-uuid"].(string); ok {
		return val
	}
	return ""
}
func (ev *Event) TS() time.Time {
	if val, ok := ev.Headers["ts"]; ok {
		switch v := val.(type) {
		case time.Time:
			return v
		case int:
			return time.Unix(int64(v), 0)
		case float64:
			return time.Unix(int64(v), int64((v-float64(int64(v)))*float64(time.Second)))
		case float32:
			return time.Unix(int64(v), int64((v-float32(int64(v)))*float32(time.Second)))
		case string:
			ts, err := time.Parse(time.RFC3339, v)
			if err == nil {
				return ts
			} else {
				return time.Time{}
			}
		default:
			return time.Time{}
		}
	} else {
		return time.Time{}
	}
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
	ev.prepared = true
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

func (ev *Event) Send(path string) error {
	if !ev.prepared {
		err := ev.Prepare()
		if err != nil {
			return err
		}
	}
	if ev.transport == nil {
		return fmt.Errorf("event has no transport")
	}
	return ev.transport.SendEvent(path, *ev)
}

func (ev *Event) Reply(reply Event) error {
	if len(ev.ReplyTo) < 1 {
		return fmt.Errorf("No reply-to header in orignal event: %+v", ev)
	}
	has := func(key string) bool { _, ok := ev.Headers[key]; return ok }
	if has("correlation-id") {
		reply.Headers["correlation-id"] = ev.Headers["correlation-id"]
	} else if has("node-name") {
		reply.Headers["correlation-id"] = ev.Headers["node-name"].(string) + "-" + time.Now().Format(time.RFC3339)
	} else {
		return fmt.Errorf("missing node-name in event")
	}
	if !reply.prepared {
		err := reply.Prepare()
		if err != nil {
			return err
		}
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
