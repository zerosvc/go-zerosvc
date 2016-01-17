package zerosvc

import (
	"encoding/json"
	"time"
)

type Heartbeat struct {
	NodeName   string             `json:"node-name"`
	NodeUUID   string             `json:"node-uuid"`
	TS         int64              `json:"ts"`
	HBInterval int                `json:"hb-interval"`
	TTL        int                `json:"ttl"`
	Services   map[string]Service `json:"services"`
}

func (node *Node) NewHeartbeat() Event {
	var hb Heartbeat
	hb.Services = make(map[string]Service)
	node.RLock()

	hb.NodeName = node.Name
	hb.NodeUUID = node.UUID
	hb.TS = time.Now().Unix()
	hb.HBInterval = node.TTL / 3
	hb.TTL = node.TTL
	for k, v := range node.Services {
		hb.Services[k] = v
	}
	node.RUnlock()
	ev := node.NewEvent()
	ev.Body, _ = json.Marshal(hb)
	ev.Prepare()
	out, _ := json.Marshal(ev)

	ev.Body = out
	ev.Prepare()
	return ev
}
