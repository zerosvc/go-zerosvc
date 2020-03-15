package zerosvc

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"

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
}

type TransportMQTTConfig struct {
	Heartbeat int
	Prefix    string
	// use custom tls.Config
	TLS bool
	// Custom tls.Config for client auth and such
	TLSConfig *tls.Config
	// Use persistent queues
}

func TransportMQTT(addr string, cfg interface{}) Transport {
	var t *trMQTT = &trMQTT{}
	var c TransportMQTTConfig
	if cfg != nil {
		c = cfg.(TransportMQTTConfig)
	}
	t.addr = addr
	t.cfg = &c
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
	if urlParsed.User != nil && urlParsed.User.Username() != "" {
		clientOpts.Username = urlParsed.User.Username()
		clientOpts.Password, _ = urlParsed.User.Password()
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
	token := t.client.Publish(path, 0, false, msg)
	// TODO add async mode
	token.Wait()
	return err
}

func (t *trMQTT) SendReply(addr string, ev Event) error {
	return nil
}

// Prepare a goroutine that will listen for incoming messages matching filter (or if empty, any) and send it to channel

func (t *trMQTT) GetEvents(filter string, channel chan Event) error {
	if token := t.client.Subscribe(filter, 0, func(client mqtt.Client, msg mqtt.Message) {
		ev := NewEvent()
		ev.transport = t
		// TODO do something about err ? send as pseudo-event ?
		err := json.Unmarshal(msg.Payload(), &ev)
		fmt.Printf("RECEIVED %+v %+v", ev, err)

		channel <- ev
	}); token.Wait() && token.Error() != nil {
		return fmt.Errorf("subscription failed: %s", token.Error())
	}
	return nil
}
