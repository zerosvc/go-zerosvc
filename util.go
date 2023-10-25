package zerosvc

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"log"
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

var base64Replacer = strings.NewReplacer(
	"+", "",
	"/", "",
	"=", "",
)

// MapBytesToTopicTitle maps binary data to topic-friendly subset of characters.
func mapBytesToTopicTitle(data []byte) string {
	str := base64.StdEncoding.EncodeToString(data)
	return base64Replacer.Replace(str)
}
func rngBlob(bytes int) []byte {
	rnd := make([]byte, bytes)
	i, err := rand.Read(rnd)
	if err == nil && i == bytes {
		return rnd
	}
	var errctr uint8
	var readctr = i
	for {
		errctr++
		if errctr > 10 {
			log.Panicf("could not get data from RNG")
		}
		i, err := rand.Read(rnd[readctr:])
		if i > 0 {
			readctr += i
		} else {
			panic(fmt.Sprintf("error getting RNG: %s", err))
		}
		if readctr >= bytes {
			return rnd
		}
	}
}
