package zerosvc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetFQDN(t *testing.T) {
	fqdn1 := GetFQDN()
	fqdn2 := GetFQDN()
	assert.NotEqual(t,"",fqdn1)
	assert.Equal(t,fqdn1,fqdn2)

}