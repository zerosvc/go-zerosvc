package zerosvc

import ()

func (node *Node) NewHeartbeat() Event {
	ev := node.NewEvent()
	return ev
}
