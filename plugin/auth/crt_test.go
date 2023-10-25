package auth

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDummy(t *testing.T) {
	a := New()
	err := a.LoadCA("../../t-data/ca-crt.pem")
	assert.NoError(t, err)
}
