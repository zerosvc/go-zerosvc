package zerosvc

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/url"

	"testing"
	"time"
)

func TestEvent(t *testing.T) {
	tr, err := NewTransportMQTTv5(ConfigMQTTv5{
		ID:      t.Name(),
		MQTTURL: []*url.URL{getTestMQURL()},
	})
	nodename := t.Name()
	node, err := NewNode(Config{
		NodeName:  nodename,
		NodeUUID:  "77ab2b23-4f1b-4247-be45-000000000010",
		Transport: tr,
		EventRoot: "test",
	})
	require.NoError(t, err)
	ev := node.NewEvent()

	t.Run("create event", func(t *testing.T) {
		assert.Equal(t, nodename, ev.NodeName)
		assert.Equal(t, "77ab2b23-4f1b-4247-be45-000000000010", ev.NodeUUID)
		assert.Equal(t, time.Time{}, ev.TS)

	})
	t.Run("prepare event", func(t *testing.T) {
		type Bo struct {
			CakeCount int    `json:"cake_count"`
			CakeType  string `json:"cake_type"`
		}
		bo := Bo{
			CakeCount: 10,
			CakeType:  "Chocolate",
		}
		err := ev.Marshal(bo)
		require.NoError(t, err)
		boout := Bo{}
		err = ev.Unmarshal(&boout)
		assert.Equal(t, bo, boout)
	})
}

func BenchmarkNewEvent(b *testing.B) {
	tr, _ := NewTransportMQTTv3(ConfigMQTTv3{
		ID:      b.Name(),
		MQTTURL: []*url.URL{getTestMQURL()},
	})
	node, _ := NewNode(Config{
		NodeName:  b.Name(),
		NodeUUID:  "77ab2b23-4f1b-4247-be45-000000000012",
		Transport: tr,
	})
	for i := 0; i < b.N; i++ {
		ev := node.NewEvent()
		ev.Marshal([]byte("test"))
	}
}
