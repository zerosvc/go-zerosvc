package zerosvc

import (
	"github.com/XANi/goneric"
	"net/url"
)

func getTestMQURL() *url.URL {
	return goneric.Must(url.Parse("tcp://guest:guest@127.0.0.1:1883"))
}
