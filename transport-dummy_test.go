package zerosvc

import (
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewTransportDummy(t *testing.T) {
	tr, err := NewTransportDummy(ConfigDummy{})
	connected := false
	require.NoError(t, tr.Connect(Hooks{
		ConnectHook:        func() { connected = true },
		ConnectionLossHook: func(e error) {},
	}, "discovery/test"))
	require.NoError(t, err)
	assert.True(t, connected)
	tdata := make([]byte, 8)
	rand.Read(tdata)
	chName := "_test/" + t.Name()
	subCh := make(chan *Message, 8)
	require.NoError(t, tr.Subscribe(chName, subCh))
	require.NoError(t, tr.Publish(Message{
		Topic:           "_test/" + t.Name(),
		ResponseTopic:   "",
		CorrelationData: nil,
		ContentType:     "",
		Metadata:        nil,
		Payload:         tdata,
		Retain:          false,
	}))
	assert.NoError(t, tr.Disconnect())
}
