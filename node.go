package zerosvc

import (
	"github.com/satori/go.uuid"
)

var namespace = `63082cd1-0f91-48cd-923a-f1523a26549b`

type Node struct {
	Name string
	UUID string
}

func NewNode(NodeName string, NodeUUID ...string) Node {
	var r Node
	r.Name = NodeName
	if len(NodeUUID) > 0 {
		r.UUID = NodeUUID[0]
	} else {
		ns, _ := uuid.FromString(namespace)
		uuid := uuid.NewV5(ns, NodeName)
		r.UUID = uuid.String()
	}
	return r
}
