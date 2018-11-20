package zerosvc

import (
	"crypto/sha512"
	"fmt"
	"strings"
)

func generatePersistentQueueName(filter string, hashkeys ...string) string {
	routingPart := filter
	if len(routingPart) > 24 {
		routingPart = routingPart[:24]
	}
	st := filter + strings.Join(hashkeys, "\000")
	h := sha512.New()
	h.Write([]byte(st))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	return strings.Join([]string{routingPart, hash[:32]}, "-")
}
