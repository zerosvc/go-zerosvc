package zerosvc

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/XANi/goneric"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"io/ioutil"
	"net/url"
	"strings"
	"time"
)

type TransportMQTTv3 struct {
	client mqtt.Client
}

// ID will be mangled to fit 23 characters if it is longer
// or replaced with client cert ID
func NewTransportMQTTv3(id string, mqttURL ...*url.URL) (*TransportMQTTv3, error) {
	tr := &TransportMQTTv3{}
	if len(mqttURL) < 1 {
		return nil, fmt.Errorf("need at least one URL")
	}
	clientOpts := mqtt.NewClientOptions()
	for _, u := range mqttURL {
		clientOpts.AddBroker(u.Scheme + "://" + u.Host)
	}
	if mqttURL[0].User != nil && mqttURL[0].User.Username() != "" {
		clientOpts.Username = mqttURL[0].User.Username()
		clientOpts.Password, _ = mqttURL[0].User.Password()
	}
	clientOpts.SetAutoReconnect(true)
	clientOpts.SetConnectRetry(true)
	clientOpts.SetConnectRetryInterval(time.Second * 10)
	clientOpts.SetMaxReconnectInterval(time.Minute)
	if mqttURL[0].Scheme == "ssl" {
		var tlsCfg tls.Config
		if len(mqttURL[0].Query().Get("cert")) > 0 {
			certPath := mqttURL[0].Query().Get("cert")
			keyPath := mqttURL[0].Query().Get("certkey")
			if len(keyPath) == 0 {
				keyPath = certPath
			}
			cert, err := tls.LoadX509KeyPair(certPath, keyPath)
			if err != nil {
				return nil, fmt.Errorf("error loading cert/key: %s", err)
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
			parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
			// make user-friendly clientId if we can parse the cert
			if err == nil {
				nameParts := strings.Split(parsedCert.Subject.CommonName, ".")
				// reverse, function later on cuts the front part.
				goneric.SliceReverseInplace(nameParts)
				id = strings.Join(nameParts, ".")
			}
		}
		if len(mqttURL[0].Query().Get("ca")) > 0 {
			certpool := x509.NewCertPool()
			pem, err := ioutil.ReadFile(mqttURL[0].Query().Get("ca"))
			if err != nil {
				return nil, fmt.Errorf("error loading CA: %s", err)
			}
			certpool.AppendCertsFromPEM(pem)
			tlsCfg.RootCAs = certpool
		}
		clientOpts.SetTLSConfig(&tlsCfg)
	}
	// MQTT3 limit
	if len(id) == 0 {
		return nil, fmt.Errorf("id is required")
	}
	if len(id) <= 23 {
		clientOpts.SetClientID(id)
	} else {
		h := sha256.New()
		h.Write([]byte(id))
		hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
		clientOpts.SetClientID(id[len(id)-12:] + hash[:11])
	}
	tr.client = mqtt.NewClient(clientOpts)
	connectToken := tr.client.Connect()
	notTimedOut := connectToken.WaitTimeout(time.Minute)
	if notTimedOut {
		return tr, connectToken.Error()
	} else {
		return nil, fmt.Errorf("timed out on connection [%s]", connectToken.Error())
	}
}

func (t *TransportMQTTv3) Publish(topic string, data []byte) error {
	token := t.client.Publish(topic, 1, true, data)
	token.Wait()
	return token.Error()
}
