package zerosvc

import (
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

func TestNewNode(t *testing.T) {
	tr, err := NewTransportMQTTv3(ConfigMQTTv3{
		ID:       t.Name(),
		WillPath: "",
		MQTTURL:  []*url.URL{getTestMQURL()},
	})
	require.NoError(t, err)
	node, err := NewNode(Config{
		NodeName:        "testNode",
		NodeUUID:        "",
		AutoHeartbeat:   false,
		Transport:       tr,
		AutoSigner:      nil,
		Signer:          nil,
		Encoder:         nil,
		Decoder:         nil,
		EventRoot:       "",
		DiscoveryPrefix: "",
	})
	require.NoError(t, err)
	ev := node.NewEvent()
	path := "_t/" + t.Name()
	require.NoError(t, node.SendEvent(path, ev))
}
