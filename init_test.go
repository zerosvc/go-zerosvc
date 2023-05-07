package zerosvc

import (
	"github.com/XANi/goneric"
	"net/url"
	"os"
)

func getTestMQURL() *url.URL {
	defaultURL := "tcp://guest:guest@127.0.0.1:1883"
	envUrl := os.Getenv("TEST_MQTT_URL")
	if len(envUrl) > 4 {
		defaultURL = envUrl
	}
	return goneric.Must(url.Parse(defaultURL))
}
