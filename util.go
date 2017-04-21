package zerosvc

import (
	"crypto/sha512"
	"fmt"
	"strings"
)

func generatePersistentQueueName(target string, routingKey string) string {
	tgtPart := target
	routingPart := routingKey
	if len(tgtPart) > 8 {
		tgtPart = tgtPart[:8]
	}
	if len(routingPart) > 16 {
		routingPart = routingPart[:16]
	}
	st := "sms-evqueue"
	h := sha512.New()
	h.Write([]byte(st))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	return strings.Join([]string{tgtPart, routingPart, hash[:16]}, "-")
}
