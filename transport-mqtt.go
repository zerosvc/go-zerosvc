package zerosvc

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
)

type trMQTT struct {
	Transport
	addr     string
	user     string
	password string
	useAuth  bool
	client   mqtt.Client
	exchange string
	cfg      *TransportMQTTConfig
	autoack  bool
	closers  []func()
	sync.Mutex
}

type TransportMQTTConfig struct {
	Heartbeat int
	Prefix    string
	// use custom tls.Config
	TLS bool
	// Custom tls.Config for client auth and such
	TLSConfig *tls.Config
	// topic to send last will message. Set to anything to enable last will
	LastWillTopic string
	// message. Sent zero + LastWillRetain if you want to clear retained message
	LastWillMessage []byte
	LastWillRetain bool
	LastWillQos uint8


}

func TransportMQTT(addr string, cfg interface{}) Transport {
	return MQTTTransport(addr, cfg.(TransportMQTTConfig))
}

func MQTTTransport(address string, config TransportMQTTConfig) Transport {
	t := &trMQTT{}
	t.addr = address
	t.cfg = &config
	t.closers = make([]func(), 0)
	return t
}

func (t *trMQTT) Connect() error {
	var err error
	// TODO URL validation should be at transport creation
	urlParsed, err := url.Parse(t.addr)
	if err != nil {
		return fmt.Errorf("Can't parse MQTT url [%s]:%s", t.addr, err)
	}
	clientOpts := mqtt.NewClientOptions().AddBroker(t.addr)
	clientOpts.OnConnectionLost = func(c mqtt.Client, err error) {
		defer func() {
			if r := recover(); r != nil {
				// ignore double close, in case we try to close something closed already by lib user
			}
		}()
		t.Lock()
		defer t.Unlock()
		for _, cl := range t.closers {
			cl()
		}
		t.closers = make([]func(), 0)

	}
	if urlParsed.User != nil && urlParsed.User.Username() != "" {
		clientOpts.Username = urlParsed.User.Username()
		clientOpts.Password, _ = urlParsed.User.Password()
	}
	if len(t.cfg.LastWillTopic) > 0 {
		clientOpts.WillEnabled=true
		clientOpts.WillTopic = t.cfg.LastWillTopic
		clientOpts.WillPayload = t.cfg.LastWillMessage
		clientOpts.WillRetained = t.cfg.LastWillRetain
		clientOpts.WillQos = t.cfg.LastWillQos
	}

	if urlParsed.Scheme == "tls" {
		var tlsCfg tls.Config
		if len(urlParsed.Query().Get("cert")) > 0 {
			certPath := urlParsed.Query().Get("cert")
			keyPath := urlParsed.Query().Get("certkey")
			if len(keyPath) == 0 {
				keyPath = certPath
			}
			cert, err := tls.LoadX509KeyPair(certPath, keyPath)
			if err != nil {
				return fmt.Errorf("error loading cert/key: %s", err)
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
			parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
			// make user-friendly clientId if we can parse the cert
			if err == nil {
				nameParts := strings.Split(parsedCert.Subject.CommonName, ".")
				// clientid max 22 characters (mqtt v3.1.1 3.1.3.1)
				if len(nameParts) > 0 {
					var partHost string
					if len(nameParts[0]) > 10 {
						partHost = nameParts[0][0:10]
					} else {
						partHost = nameParts[0]
					}
					r := make([]byte, 10)
					i, err := rand.Read(r)
					// fallboack to no/server generated clientid
					if i == len(r) && err == nil {
						h := sha256.New()
						h.Write(r)
						partRand := mapBytesToMQTTAlpha(h.Sum(nil))
						clientOpts.ClientID = partHost + partRand[0:(22-len(partHost))]

					}
				}
			}
		}
		if len(urlParsed.Query().Get("ca")) > 0 {
			certpool := x509.NewCertPool()
			pem, err := ioutil.ReadFile(urlParsed.Query().Get("ca"))
			if err != nil {
				return fmt.Errorf("error loading CA: %s", err)
			}
			certpool.AppendCertsFromPEM(pem)
			tlsCfg.RootCAs = certpool
		}
		clientOpts.SetTLSConfig(&tlsCfg)
	}
	t.client = mqtt.NewClient(clientOpts)
	if connectToken := t.client.Connect(); connectToken.Wait() && connectToken.Error() != nil {
		return fmt.Errorf("Could not connect to MQTT: %s", connectToken.Error())
	}

	return nil
}
func (t *trMQTT) Shutdown() {
	// TODO expose as config
	t.client.Disconnect(1000)
}

func prepareMQTTMsg(ev *Event) ([]byte, error) {
	return json.Marshal(ev)
}

func (t *trMQTT) SendEvent(path string, ev Event) error {
	msg, err := prepareMQTTMsg(&ev)
	if err != nil {
		return err
	}
	retained := false
	if ev.RetainTill != nil && ev.RetainTill.After(time.Now()) {
		retained = true
	}
	token := t.client.Publish(path, 0, retained, msg)
	// TODO add async mode
	token.Wait()
	return token.Error()
}

func (t *trMQTT) SendReply(path string, ev Event) error {
	ev.ReplyTo = ""
	return t.SendEvent(path, ev)
}

// Prepare a goroutine that will listen for incoming messages matching filter (or if empty, any) and send it to channel
func (t *trMQTT) GetEvents(filter string, channel chan Event) error {
	if token := t.client.Subscribe(filter, 0, func(client mqtt.Client, msg mqtt.Message) {
		// overhead is tiny as of go-1.14 and it allows client to close channel without this side panicking
		defer func() {
			if recover() != nil {
				t.client.Unsubscribe(filter)
			}
		}()
		ev := NewEvent()
		ev.transport = t
		// TODO do something about err ? send as pseudo-event ?
		err := json.Unmarshal(msg.Payload(), &ev)
		_ = err
		ev.RoutingKey = msg.Topic()

		channel <- ev
	}); token.Wait() && token.Error() != nil {
		close(channel)
		return fmt.Errorf("subscription failed: %s", token.Error())
	}
	t.Lock()
	t.closers = append(t.closers, func() {
		close(channel)
	})
	t.Unlock()
	return nil
}
