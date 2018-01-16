package zerosvc

import "fmt"

// New is helper for common use case. Create node, add transport to it, connect to transport, return node and error
// et
func New(NodeName string, Transport Transport) (*Node, error) {
	err := Transport.Connect()
	if err != nil {
		return nil, fmt.Errorf("Error when connecting to transport: %s", err)
	}
	node := NewNode(NodeName)
	node.SetTransport(Transport)
	return node, nil

}
