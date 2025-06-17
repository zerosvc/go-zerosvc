package zerosvc

import (
	mqtt "github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"go.uber.org/zap"
)

type TransportDummy struct {
	mqttCfg  mqtt.ClientConfig
	willPath string
	l        *zap.SugaredLogger
}

type ConfigDummy struct {
	Logger *zap.SugaredLogger
}

func NewTransportDummy(cfg ConfigDummy) (*TransportDummy, error) {
	mqttTr := &TransportDummy{
		l: cfg.Logger,
	}
	if mqttTr.l == nil {
		mqttTr.l = zap.NewNop().Sugar()
	}
	return mqttTr, nil
}
func (t *TransportDummy) Connect(h Hooks, willPath string) error {
	if h.ConnectHook != nil {
		t.mqttCfg.OnConnectionUp = func(cm *mqtt.ConnectionManager, ca *paho.Connack) {
			h.ConnectHook()
		}
		h.ConnectHook()
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
	return nil
}
func (t *TransportDummy) Publish(m Message) error {
	return nil
}

func (t *TransportDummy) Subscribe(topic string, data chan *Message) error {
	return nil
}

func (t *TransportDummy) HeartbeatMessage(m Message) error {
	m.Retain = true
	m.Topic = t.willPath
	return t.Publish(m)
}

func (t *TransportDummy) Disconnect() error {
	return nil
}
