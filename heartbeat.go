package zerosvc

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

type Heartbeat struct {
	NodeName   string                 `json:"node-name"`
	NodeUUID   string                 `json:"node-uuid"`
	NodeInfo   map[string]interface{} `json:"node-info,omitempty"`
	NodePubkey string                 `json:"node-pubkey,omitempty"`
	TS         int64                  `json:"ts"`
	HBInterval int                    `json:"hb-interval"`
	TTL        int                    `json:"ttl"`
	Services   map[string]Service     `json:"services"`
}

func (n *Node) NewHeartbeat() Event {
	var hb Heartbeat
	hb.Services = make(map[string]Service)
	n.RLock()

	hb.NodeName = n.Name
	hb.NodeUUID = n.UUID
	hb.TS = time.Now().Unix()
	hb.HBInterval = int(n.TTL.Seconds()) / 3
	hb.TTL = int(n.TTL.Seconds())
	hb.NodeInfo = n.Info
	for k, v := range n.Services {
		hb.Services[k] = v
	}
	if n.Signer != nil {
		pubkeyPkt := make([]byte, 2)
		pubkeyPkt[0] = n.Signer.Type()
		key := n.Signer.PublicKey()
		pubkeyPkt[1] = uint8(len(key))
		pubkeyPkt = append(pubkeyPkt, key...)
		hb.NodePubkey = base64.StdEncoding.EncodeToString(pubkeyPkt)
	}
	n.RUnlock()
	ev := n.NewEvent()
	validity := time.Now().Add(n.TTL)
	ev.RetainTill = &validity
	ev.Body, _ = json.Marshal(hb)
	ev.Prepare()
	return ev
}

// Heartbeater runs heartbeat. Run in goroutine
func (n *Node) Heartbeater(path ...string) {
	for {
		ev := n.NewHeartbeat()
		if len(path) == 0 {
			n.SendEvent(n.HeartbeatPath(), ev)
		} else {
			n.SendEvent(path[0], ev)
		}

		time.Sleep(n.TTL / 3)
	}
}
