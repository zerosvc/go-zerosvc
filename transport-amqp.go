package zerosvc

import (
	"github.com/streadway/amqp"
	"os"
	"time"
	"crypto/tls"
	"strings"
)

type trAMQP struct {
	Transport
	addr string
	Conn *amqp.Connection
	exchange string
	cfg *TransportAMQPConfig
}

type TransportAMQPConfig struct {
	Heartbeat int
	EventExchange string
	// use custom tls.Confg ( will still try default if amqps:// is specified
	TLS bool
	// Custom tls.Config for client auth and such
	TLSConfig *tls.Config
}

func TransportAMQP(addr string, cfg interface{}) Transport {
	var t *trAMQP = &trAMQP{}
	var c TransportAMQPConfig
	if cfg != nil {
		c = cfg.(TransportAMQPConfig)
	}
	t.addr = addr
	t.cfg = &c
	if len(c.EventExchange) > 0 {
		t.exchange = c.EventExchange
	} else {
		t.exchange = "events"
	}
	return t
}

func (t *trAMQP) Connect() error {
	var err error
	var conn *amqp.Connection
	if strings.Contains(t.addr, "amqps") && t.cfg.TLS {
		conn, err = amqp.DialTLS(t.addr, t.cfg.TLSConfig)
	} else {
		conn, err = amqp.Dial(t.addr)
	}
	if err != nil {
		return err
	}
	t.Conn = conn
	return err
}
func (t *trAMQP) Shutdown() {
	t.Conn.Close()
}

func prepareAMQPMsg(ev *Event) amqp.Publishing {
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body:         ev.Body,
	}
	has := func(key string) bool { _, ok := ev.Headers[key]; return ok }
	if has("node-name") {
		msg.AppId = ev.Headers["node-name"].(string)
		delete(ev.Headers, "node-name")
	}
	if has("correlation-id") {
		msg.CorrelationId = ev.Headers["correlation-id"].(string)
		delete(ev.Headers, "correlation-id")
	}
	if has("user-id") {
		msg.UserId = ev.Headers["user-id"].(string)
		delete(ev.Headers, "user-id")
	}
	msg.Headers = ev.Headers
	return msg
}

func (t *trAMQP) SendEvent(path string, ev Event) error {
	ch, err := t.Conn.Channel()
	defer ch.Close()
	if err != nil {
		return err
	}
	msg := prepareAMQPMsg(&ev)
	err = ch.Publish(t.exchange, path, false, false, msg)
	return err
}

func (t *trAMQP) SendReply(addr string, ev Event) error {
	var e error
	ch, err := t.Conn.Channel()
	if err != nil {
		return err
	}
	msg := prepareAMQPMsg(&ev)
	// "" is default exchange that should route to queue specified in path
	err = ch.Publish("", addr, false, false, msg)
	return e

}

// Prepare a goroutine that will listen for incoming messages matching filter (or if empty, any) and send it to channel

func (t *trAMQP) GetEvents(filter string, channel chan Event) error {
	ch, err := t.Conn.Channel()
	if err != nil {
		return err
	}
	if filter == "" {
		filter = "#"
	}
	q, err := t.amqpCreateAndBindQueue(ch, filter)
	// we only try to create exchange to not take the cost on checking if exchange exists on every connection
	if err != nil {
		err = t.amqpCreateEventsExchange()
		if err != nil {
			return err
		} else {
			ch, err = t.Conn.Channel()
			q, err = t.amqpCreateAndBindQueue(ch, filter)
			if err != nil {
				return err
			}
		}
	}
	go t.amqpEventReceiver(ch, q, channel)
	return err
}

func (t *trAMQP) amqpCreateAndBindQueue(ch *amqp.Channel, filter string) (amqp.Queue, error) {
	q, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when usused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return q, err
	}
	err = ch.QueueBind(
		q.Name,   // queue name
		filter,   // routing key
		t.exchange, // exchange
		false,
		nil,
	)
	if err != nil {
		return q, err
	}
	return q, err
}

func (t *trAMQP) amqpEventReceiver(ch *amqp.Channel, q amqp.Queue, c chan Event) {
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		//fixme send error to something ?
	}
	for d := range msgs {
		var ev Event
		ev.transport = t
		ev.Headers = d.Headers
		ev.Headers["_transport-exchange"] = d.Exchange
		ev.Headers["_transport-RoutingKey"] = d.RoutingKey
		ev.Headers["_transport-ContentType"] = d.ContentType
		has := func(key string) bool { _, ok := ev.Headers[key]; return ok }
		if len(d.AppId) > 0 && !has("node-name") {
			ev.Headers["node-name"] = d.AppId
		}
		if len(d.CorrelationId) > 0 && !has("correlation-id") {
			ev.Headers["correlation-id"] = d.CorrelationId
		}
		if len(d.ReplyTo) > 0 {
			ev.ReplyTo = d.ReplyTo
		}
		ev.Body = d.Body
		c <- ev
	}
	c <- Event{
		Body: []byte("dc1?"),
	}
	os.Exit(1)
}

// **DESTRUCTIVE**
//
// remove all used exchanges
//
// mostly used for cleanup before tests
func (t *trAMQP) AdminCleanup() {
	ch, err := t.Conn.Channel()
	ch.ExchangeDelete("events", false, false)
	_ = err
}
func (t *trAMQP) amqpCreateEventsExchange() error {
	ch, err := t.Conn.Channel()
	if err != nil {
		return err
	}

	err = ch.ExchangeDeclare(
		t.exchange, // name
		"topic",  // type
		true,     // durable
		false,    // autoDelete
		false,    // internal
		false,    // no wait
		nil,
	)
	return err
}
