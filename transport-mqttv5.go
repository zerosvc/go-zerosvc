package zerosvc

import (
	"context"
	"crypto/tls"
	"fmt"
	mqtt "github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"go.uber.org/zap"
	"net/url"
	"time"
)

type TransportMQTTv5 struct {
	mqttCtx  context.Context
	mqttCfg  mqtt.ClientConfig
	client   *mqtt.ConnectionManager
	router   paho.Router
	timeout  time.Duration
	willPath string
	l        *zap.SugaredLogger
}

type ConfigMQTTv5 struct {
	ID      string
	MQTTURL []*url.URL
	Logger  *zap.SugaredLogger
}

func NewTransportMQTTv5(cfg ConfigMQTTv5) (*TransportMQTTv5, error) {
	var tlsC *tls.Config
	mqttTr := &TransportMQTTv5{
		mqttCtx: context.Background(),
		timeout: time.Second * 30, // # TODO config
		l:       cfg.Logger,
	}
	if mqttTr.l == nil {
		mqttTr.l = zap.NewNop().Sugar()
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
		ClientID: sanitizeClientID(cfg.ID),
		Router:   mqttTr.router,
	}
	mqttTr.mqttCfg = mqtt.ClientConfig{
		ServerUrls:        cfg.MQTTURL,
		TlsCfg:            tlsC,
		KeepAlive:         30,
		ConnectRetryDelay: 10 * time.Second,
		ConnectTimeout:    30 * time.Second,
		ClientConfig:      clientMqttConfig,
		ConnectPacketBuilder: func(connect *paho.Connect, u *url.URL) (*paho.Connect, error) {
			if u.User != nil && len(u.User.Username()) > 0 {
				connect.Username = u.User.Username()
			}
			return connect, nil
		},
	}

	if cfg.MQTTURL[0].User != nil && len(cfg.MQTTURL[0].User.Username()) > 0 {
		mqttTr.mqttCfg.ConnectUsername = cfg.MQTTURL[0].User.Username()
		p, _ := cfg.MQTTURL[0].User.Password()
		mqttTr.mqttCfg.ConnectPassword = []byte(p)
	}
	return mqttTr, nil
}
func (t *TransportMQTTv5) Connect(h Hooks, willPath string) error {
	if len(willPath) == 0 {
		return fmt.Errorf("will path must be set")
	}
	t.mqttCfg.SetWillMessage(
		willPath,
		[]byte(""),
		1,
		true,
	)
	t.willPath = willPath
	if h.ConnectHook != nil {
		t.mqttCfg.OnConnectionUp = func(cm *mqtt.ConnectionManager, ca *paho.Connack) {
			h.ConnectHook()
		}
	} else {
		t.mqttCfg.OnConnectionUp = func(cm *mqtt.ConnectionManager, ca *paho.Connack) {
			t.l.Debugf("connected")
		}
	}
	t.mqttCfg.OnClientError = func(err error) {
		t.l.Errorf("client error: %s", err)
	}
	t.mqttCfg.OnConnectError = func(err error) {
		t.l.Errorf("error connecting: %s, %T", err, err)
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
		Retain:   m.Retain,
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
		return fmt.Errorf("pub %w:", err)
	}
	_ = resp
	return nil
}

func (t *TransportMQTTv5) Subscribe(topic string, data chan *Message) error {
	subTimeout, _ := context.WithTimeout(t.mqttCtx, t.timeout)
	sub := &paho.Subscribe{
		Properties: nil,
		Subscriptions: []paho.SubscribeOptions{
			{
				Topic:             topic,
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

func (t *TransportMQTTv5) HeartbeatMessage(m Message) error {
	m.Retain = true
	m.Topic = t.willPath
	return t.Publish(m)
}

func (t *TransportMQTTv5) Disconnect() error {
	return t.client.Disconnect(t.mqttCtx)
}
