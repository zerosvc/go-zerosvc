package zerosvc

import (
	"context"
	"crypto/tls"
	"fmt"
	mqtt "github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"net/url"
	"time"
)

type TransportMQTTv5 struct {
	mqttCtx context.Context
	mqttCfg mqtt.ClientConfig
	client  *mqtt.ConnectionManager
	router  paho.Router
	timeout time.Duration
}

type ConfigMQTTv5 struct {
	ID       string
	WillPath string
	MQTTURL  []*url.URL
}

func NewTransportMQTTv5(cfg ConfigMQTTv5) (*TransportMQTTv5, error) {
	var tlsC *tls.Config
	mqttTr := &TransportMQTTv5{
		mqttCtx: context.Background(),
		timeout: time.Second * 30, // # TODO config
	}
	if cfg.MQTTURL[0].Scheme == "ssl" {
		var subject string
		var err error
		tlsC, subject, err = getTLSConfigFromURL(cfg.MQTTURL[0])
		if err != nil {
			return nil, fmt.Errorf("error loading certs: %s", err)
		}
		if len(cfg.ID) < 1 {
			cfg.ID = subject
		}
	}
	if len(cfg.ID) == 0 {
		return nil, fmt.Errorf("clientID can't be empty")
	}
	mqttTr.router = paho.NewStandardRouter()
	clientMqttConfig := paho.ClientConfig{
		ClientID:                   sanitizeClientID(cfg.ID),
		Conn:                       nil,
		MIDs:                       nil,
		AuthHandler:                nil,
		PingHandler:                nil,
		Router:                     mqttTr.router,
		Persistence:                nil,
		PacketTimeout:              0,
		OnServerDisconnect:         nil,
		OnClientError:              nil,
		PublishHook:                nil,
		EnableManualAcknowledgment: false,
		SendAcksInterval:           0,
	}
	mqttTr.mqttCfg = mqtt.ClientConfig{
		BrokerUrls:        cfg.MQTTURL,
		TlsCfg:            tlsC,
		KeepAlive:         30,
		ConnectRetryDelay: 10 * time.Second,
		ConnectTimeout:    30 * time.Second,
		WebSocketCfg:      nil,
		OnConnectionUp:    nil,
		OnConnectError:    nil,
		Debug:             nil,
		PahoDebug:         nil,
		PahoErrors:        nil,
		ClientConfig:      clientMqttConfig,
	}
	return mqttTr, nil
}
func (t *TransportMQTTv5) Connect(h Hooks) error {
	if h.ConnectHook != nil {
		t.mqttCfg.OnConnectionUp = func(cm *mqtt.ConnectionManager, ca *paho.Connack) {
			h.ConnectHook()
		}
	}
	conn, err := mqtt.NewConnection(t.mqttCtx, t.mqttCfg)
	if err != nil {
		return err
	}
	connectCtx, _ := context.WithTimeout(t.mqttCtx, time.Second*30)
	if err = conn.AwaitConnection(connectCtx); err != nil {
		return fmt.Errorf("error connecting to mq: %w", err)
	}
	t.client = conn
	return nil
}
func (t *TransportMQTTv5) Publish(m Message) error {
	pubTimeout, _ := context.WithTimeout(t.mqttCtx, t.timeout)
	ev := &paho.Publish{
		PacketID: 0,
		QoS:      0,
		Retain:   false,
		Topic:    m.Topic,
		Properties: &paho.PublishProperties{
			CorrelationData:        m.CorrelationData,
			ContentType:            m.ContentType,
			ResponseTopic:          m.ResponseTopic,
			PayloadFormat:          nil,
			MessageExpiry:          nil,
			SubscriptionIdentifier: nil,
			TopicAlias:             nil,
			User:                   paho.UserProperties{},
		},
		Payload: m.Payload,
	}
	resp, err := t.client.Publish(pubTimeout, ev)
	if err != nil {
		//return fmt.Errorf("pub %w: %s[%d]", err, resp.Properties.ReasonString, resp.ReasonCode)
		fmt.Printf("pub %s:\n", err)
		return fmt.Errorf("pub %w:", err)
	}
	_ = resp
	return nil
}

func (t *TransportMQTTv5) Subscribe(topic string, data chan *Message) error {
	subTimeout, _ := context.WithTimeout(t.mqttCtx, t.timeout)
	sub := &paho.Subscribe{
		Properties: nil,
		Subscriptions: map[string]paho.SubscribeOptions{
			topic: {
				QoS:               2, // TODO not sure
				RetainHandling:    0,
				NoLocal:           false,
				RetainAsPublished: false,
			},
		},
	}
	suback, err := t.client.Subscribe(subTimeout, sub)
	if err != nil {
		return fmt.Errorf("sub %w: %s[%+v]", err, suback.Properties.ReasonString, suback.Reasons)
	}
	t.router.RegisterHandler(topic, func(p *paho.Publish) {
		msg := Message{
			Topic:           p.Topic,
			ResponseTopic:   p.Properties.ResponseTopic,
			CorrelationData: p.Properties.CorrelationData,
			ContentType:     p.Properties.ContentType,
			Metadata:        map[string]string{},
			Payload:         p.Payload,
		}
		if len(p.Properties.User) > 0 {
			for _, prop := range p.Properties.User {
				msg.Metadata[prop.Key] = prop.Value
			}
		}
		data <- &msg
	})
	return nil
}

func (t *TransportMQTTv5) Disconnect() error {
	return t.client.Disconnect(t.mqttCtx)
}
