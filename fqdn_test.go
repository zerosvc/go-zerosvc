package zerosvc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetFQDN(t *testing.T) {
	fqdn := GetFQDN()
	assert.NotEqual(t,"",fqdn)
}