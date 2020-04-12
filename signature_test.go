package zerosvc

import "testing"
import "github.com/stretchr/testify/assert"
import "github.com/stretchr/testify/require"

var blob = []byte("data blob to sign")

func TestSigEd25519(t *testing.T) {
	sig, err := NewSignerEd25519()
	require.NoError(t, err)
	signature1 := sig.Sign(blob)
	signature2 := sig.Sign(blob)
	t.Run("signing", func(t *testing.T) {
		assert.NotEqual(t, signature1, make([]byte, len(signature1)))
	})
	t.Run("validating", func(t *testing.T) {
		assert.True(t, sig.Verify(blob, signature1))
		assert.True(t, sig.Verify(blob, signature2))
		signature1[0]++
		signature2[4]++
		assert.False(t, sig.Verify(blob, signature1))
		assert.False(t, sig.Verify(blob, signature2))
	})
}

func TestSignedEvent(t *testing.T) {

}