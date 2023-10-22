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
	client     mqtt.Client
	clientOpts *mqtt.ClientOptions
}

type ConfigMQTTv3 struct {
	ID      string
	MQTTURL []*url.URL
}

// ID will be mangled to fit 23 characters if it is longer
// or replaced with client cert ID
// willPath points to path where the retain=true empty message will be sent on disconnect
// to be used with heartbeats to auto-clear presence informatio
func NewTransportMQTTv3(cfg ConfigMQTTv3) (*TransportMQTTv3, error) {
	tr := &TransportMQTTv3{}
	if cfg.MQTTURL == nil || len(cfg.MQTTURL) < 1 {
		return nil, fmt.Errorf("need at least one URL")
	}
	tr.clientOpts = mqtt.NewClientOptions()
	for _, u := range cfg.MQTTURL {
		tr.clientOpts.AddBroker(u.Scheme + "://" + u.Host)
	}
	if cfg.MQTTURL[0].User != nil && cfg.MQTTURL[0].User.Username() != "" {
		tr.clientOpts.Username = cfg.MQTTURL[0].User.Username()
		tr.clientOpts.Password, _ = cfg.MQTTURL[0].User.Password()
	}
	tr.clientOpts.SetAutoReconnect(true)
	tr.clientOpts.SetConnectRetry(true)
	tr.clientOpts.SetConnectRetryInterval(time.Second * 10)
	tr.clientOpts.SetMaxReconnectInterval(time.Minute)

	if cfg.MQTTURL[0].Scheme == "ssl" {
		var tlsCfg tls.Config
		if len(cfg.MQTTURL[0].Query().Get("cert")) > 0 {
			certPath := cfg.MQTTURL[0].Query().Get("cert")
			keyPath := cfg.MQTTURL[0].Query().Get("certkey")
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
				cfg.ID = strings.Join(nameParts, ".")
			}
		}
		if len(cfg.MQTTURL[0].Query().Get("ca")) > 0 {
			certpool := x509.NewCertPool()
			pem, err := ioutil.ReadFile(cfg.MQTTURL[0].Query().Get("ca"))
			if err != nil {
				return nil, fmt.Errorf("error loading CA: %s", err)
			}
			certpool.AppendCertsFromPEM(pem)
			tlsCfg.RootCAs = certpool
		}
		tr.clientOpts.SetTLSConfig(&tlsCfg)
	}
	// MQTT3 limit
	if len(cfg.ID) == 0 {
		return nil, fmt.Errorf("id is required")
	}
	if len(cfg.ID) <= 23 {
		tr.clientOpts.SetClientID(cfg.ID)
	} else {
		h := sha256.New()
		h.Write([]byte(cfg.ID))
		hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
		tr.clientOpts.SetClientID(cfg.ID[len(cfg.ID)-12:] + hash[:11])
	}

	return tr, nil
}
func (t *TransportMQTTv3) Connect(h Hooks, willPath string) error {
	//if len(willPath) > 0 { // running with empty will path will cause client to timeout
	fmt.Printf("will path: %s\n", willPath)
	t.clientOpts.SetBinaryWill(willPath, []byte{}, 1, true)
	//}
	if h.ConnectHook != nil {
		t.clientOpts.SetOnConnectHandler(func(c mqtt.Client) {
			h.ConnectHook()
		})
	}

	if h.ConnectionLossHook != nil {
		t.clientOpts.SetConnectionLostHandler(
			func(c mqtt.Client, err error) {
				h.ConnectionLossHook(err)
			})
	}
	t.client = mqtt.NewClient(t.clientOpts)

	connectToken := t.client.Connect()
	notTimedOut := connectToken.WaitTimeout(time.Minute)
	if notTimedOut {
		return connectToken.Error()
	} else {
		return fmt.Errorf("timed out on connection [%s]", connectToken.Error())
	}
}

func (t *TransportMQTTv3) Publish(m Message) error {
	token := t.client.Publish(m.Topic, 1, m.Retain, m.Payload)
	token.Wait()
	return token.Error()
}

func (t *TransportMQTTv3) Subscribe(topic string, data chan *Message) error {
	cb := func(client mqtt.Client, msg mqtt.Message) {
		m := Message{
			Topic:   msg.Topic(),
			Payload: msg.Payload(),
		}
		data <- &m
	}
	token := t.client.Subscribe(topic, 1, cb)
	token.Wait()
	return token.Error()
}
func (t *TransportMQTTv3) SetConnectHandler(topic string, data []byte) {
}

func (t *TransportMQTTv3) Disconnect() {
	if t.client != nil {
		t.client.Disconnect(10000)
	}
	// make sure it is NOT reused.
	t.client = nil
}
