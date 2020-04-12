package zerosvc

import (
	"crypto/sha512"
	"encoding/base64"
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

// only alphanum for MQTT standard
var base64MQTTAlpha = strings.NewReplacer(
	"+", "1",
	"/", "2",
)

// MapBytesToTopicTitle maps binary data to topic-friendly subset of characters.
func mapBytesToMQTTAlpha(data []byte) string {
	str := base64.StdEncoding.EncodeToString(data)
	return base64MQTTAlpha.Replace(strings.Trim(str, "="))
}
