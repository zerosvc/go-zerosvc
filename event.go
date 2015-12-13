package zerosvc

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

type Event struct {
	Headers map[string]interface{}
	Body    []byte
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
