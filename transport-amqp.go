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
	autoack bool
}

type TransportAMQPConfig struct {
	Heartbeat int
	EventExchange string
	// if set to false each event will require acknowledge or cancellation(requeue) via Ack()/Noack() methods
	NoAutoAck bool
	// use custom tls.Confg ( will still try default if amqps:// is specified
	TLS bool
	// Custom tls.Config for client auth and such
	TLSConfig *tls.Config
	// Use persistent queues
	PersistentQueue bool
	// generate queue name based on filter so load can be shares between multiple clients
	// and persisted between restart
	SharedQueue bool
	// queue TTL if using persistent ones. Do not set to never delete
	QueueTTL int64
	// message TTL in queue
	MessageTTL int64
	// dead letter exchange
	DeadLetterExchange string
	// queue prefix for non-automatic ones
	QueuePrefix string

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
	t.autoack = ! c.NoAutoAck
	if c.SharedQueue {
		c.PersistentQueue = true
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
	queueName := ""
	if t.cfg.SharedQueue {
		queueName = t.cfg.QueuePrefix + generatePersistentQueueName(t.cfg.EventExchange, filter)
	}
	if err != nil {
		return err
	}
	if filter == "" {
		filter = "#"
	}
	q, err := t.amqpCreateAndBindQueue(ch, filter, queueName)
	// we only try to create exchange to not take the cost on checking if exchange exists on every connection
	if err != nil {
		err = t.amqpCreateEventsExchange()
		if err != nil {
			return err
		} else {
			ch, err = t.Conn.Channel()
			q, err = t.amqpCreateAndBindQueue(ch, filter, queueName)
			if err != nil {
				return err
			}
		}
	}
	go t.amqpEventReceiver(ch, q, channel,t.autoack)
	return err
}

func (t *trAMQP) amqpCreateAndBindQueue(ch *amqp.Channel, filter string, queueName string) (amqp.Queue, error) {
	exclusiveQueue := true
	durableQueue := false
	queueOpts := make(amqp.Table)

	if len(queueName) > 0 {
		exclusiveQueue = false
		durableQueue = true
		if t.cfg.QueueTTL > 0 {
			queueOpts["x-expires"] = t.cfg.QueueTTL
		}
	}
	if t.cfg.MessageTTL > 0 {
		queueOpts["x-message-ttl"] = t.cfg.MessageTTL
	}
	if len(t.cfg.DeadLetterExchange) > 0 {
		queueOpts["x-dead-letter-exchange"] = t.cfg.DeadLetterExchange
	}
	q, err := ch.QueueDeclare(
		queueName,    // name
		durableQueue, // durable
		false, // delete when unused
		exclusiveQueue,  // exclusive
		false, // no-wait
		queueOpts,   // arguments
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

func (t *trAMQP) amqpEventReceiver(ch *amqp.Channel, q amqp.Queue, c chan Event,autoack bool) {
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		autoack,   // auto-ack
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
		if !autoack {
			ev.NeedsAck = true
			ev.ack = make(chan bool)
			go func(ackCh *chan bool, delivery *amqp.Delivery) {
				ackDelivery :=<- *ackCh
				if ackDelivery {
					delivery.Ack(true)
				} else {
					delivery.Nack(false, true)
				}
			} (&ev.ack,&d)
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
