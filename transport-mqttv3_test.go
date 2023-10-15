package zerosvc

import (
	"crypto/rand"
	"github.com/XANi/goneric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
	"time"
)

func TestNewTransportMQTTV3(t *testing.T) {
	tr, err := NewTransportMQTTv3(ConfigMQTTv3{
		ID:       t.Name(),
		WillPath: "",
		MQTTURL:  []*url.URL{getTestMQURL()},
	})
	connected := false
	require.NoError(t, tr.Connect(Hooks{
		ConnectHook:        func() { connected = true },
		ConnectionLossHook: func(e error) {},
	}))
	require.NoError(t, err)
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
	ret := goneric.ChanToSliceNTimeout(subCh, 1, time.Second*5)
	assert.Len(t, ret, 1)
	assert.Equal(t, tdata, ret[0].Payload)
	assert.True(t, connected)
	require.NoError(t, err)
	tr.Disconnect()

	tr2, err := NewTransportMQTTv3(ConfigMQTTv3{
		ID:       "very-long-id-name-that-exceeds-mqtt-clientid-length",
		WillPath: "",
		MQTTURL:  []*url.URL{getTestMQURL()},
	})
	require.NoError(t, err)
	require.NoError(t, tr2.Connect(Hooks{}))
	tr2.Disconnect()
	assert.Panics(t, func() {
		tr2.Subscribe(chName, subCh)
	})
}
