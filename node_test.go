package zerosvc

import (
	//	"bufio"
	//	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/url"
	"time"

	//	"os"
	//	"strings"
	"testing"
)

func TestNode(t *testing.T) {
	tr, err := NewTransportMQTTv3(ConfigMQTTv3{
		ID:      t.Name(),
		MQTTURL: []*url.URL{getTestMQURL()},
	})
	require.NoError(t, err)
	node, err := NewNode(Config{
		NodeName:  "node-" + t.Name(),
		NodeUUID:  "77ab2b23-4f1b-4247-be45-000000000000",
		Transport: tr,
		EventRoot: "test",
	})
	require.NoError(t, err)
	assert.Equal(t, "node-"+t.Name(), node.Name)
	assert.Equal(t, "77ab2b23-4f1b-4247-be45-000000000000", node.UUID)

	node3, err := NewNode(Config{
		NodeName:  "testnode",
		Transport: tr,
	})
	require.NoError(t, err)
	node4, err := NewNode(
		Config{
			NodeName:  "testnode",
			Transport: tr,
		})
	require.NoError(t, err)
	assert.Equal(t, node3.UUID, node4.UUID)
	evCh, err := node4.GetEventsCh("t4/#")
	require.NoError(t, err)
	ev := node4.NewEvent()
	ev.Body = []byte("cake")
	node4.SendEvent("t4/cake", ev)
	select {
	case <-time.After(time.Second):
		assert.True(t, false, "receiving message timed out")
	case ev := <-evCh:
		assert.Equal(t, ev.Body, []byte("cake"))
	}
}
