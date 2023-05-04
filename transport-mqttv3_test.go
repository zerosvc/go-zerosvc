package zerosvc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewTransportMQTTV3(t *testing.T) {

	_, err := NewTransportMQTTv3("test", getTestMQURL())
	require.NoError(t, err)

}
