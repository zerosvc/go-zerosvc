package zerosvc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNew(t *testing.T) {
	type cfg struct {
		Heartbeat int
	}

	node, err := NewNode(Config{
		NodeName:        "",
		NodeUUID:        "",
		Transport:       Transport{},
		AutoHeartbeat:   false,
		AutoSigner:      nil,
		Signer:          nil,
		Encoder:         nil,
		EventRoot:       "",
		DiscoveryPrefix: "",
	}, Transport{})
	require.NoError(t, err)
	_ = node
}
