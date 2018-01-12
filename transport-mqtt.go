package zerosvc

import (
	"crypto/tls"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/streadway/amqp"
	"net/url"
	"strings"
	"sync"
	"time"
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
	Heartbeat     int
	Prefix string
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
		clientOpts.Password,_ = urlParsed.User.Password()
	}
	t.client = mqtt.NewClient(clientOpts)
	if connectToken := client.Connect(); connectToken.Wait() && connectToken.Error() != nil {
		return fmt.Errorf("Could not connect to MQTT: %s", connectToken.Error())
	}
	return nil
}
func (t *trMQTT) Shutdown() {
	// TODO expose as config
	t.client.Disconnect(1000)
}

func prepareMQTTMsg(ev *Event) []byte {
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
	if has("_transport-ttl") {
		// TODO coerce any ints into string (ms)
		msg.Expiration = ev.Headers["_transport-ttl"].(string)
	}
	msg.Headers = ev.Headers
	return msg
}

func (t *trMQTT) SendEvent(path string, ev Event) error {
	ch, err := t.Conn.Channel()
	defer ch.Close()
	if err != nil {
		return err
	}
	msg := prepareMQTTMsg(&ev)
	err = ch.Publish(t.exchange, path, false, false, msg)
	return err
}

func (t *trMQTT) SendReply(addr string, ev Event) error {
	var e error
	ch, err := t.Conn.Channel()
	if err != nil {
		return err
	}
	msg := prepareMQTTMsg(&ev)
	// "" is default exchange that should route to queue specified in path
	err = ch.Publish("", addr, false, false, msg)
	return e

}

// Prepare a goroutine that will listen for incoming messages matching filter (or if empty, any) and send it to channel

func (t *trMQTT) GetEvents(filter string, channel chan Event) error {
	ch, err := t.Conn.Channel()
	if t.cfg.MaxInFlightRecv > 0 {
		ch.Qos(t.cfg.MaxInFlightRecv, 0, false)
	}
	// errch := make(chan *amqp.Error)
	// t.Conn.NotifyClose(errch)
	// ch.NotifyClose(errch)
	// go func(bb *chan *amqp.Error) {
	// 	fmt.Println(" ----- WAITING FOR ERRORS ----- ")
	// 	for z := range *bb {
	// 		panic(fmt.Sprintf("%+v",z))
	// 	}
	// }(&errch)

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
	go t.amqpEventReceiver(ch, q, channel, t.autoack)
	return err
}

func (t *trMQTT) amqpCreateAndBindQueue(ch *amqp.Channel, filter string, queueName string) (amqp.Queue, error) {
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
		queueName,      // name
		durableQueue,   // durable
		false,          // delete when unused
		exclusiveQueue, // exclusive
		false,          // no-wait
		queueOpts,      // arguments
	)
	if err != nil {
		return q, err
	}
	err = ch.QueueBind(
		q.Name,     // queue name
		filter,     // routing key
		t.exchange, // exchange
		false,
		nil,
	)
	if err != nil {
		return q, err
	}
	return q, err
}

func (t *trMQTT) amqpEventReceiver(ch *amqp.Channel, q amqp.Queue, c chan Event, autoack bool) {
	msgs, err := ch.Consume(
		q.Name,  // queue
		"",      // consumer
		autoack, // auto-ack
		false,   // exclusive
		false,   // no-local
		false,   // no-wait
		nil,     // args
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
		ev.Redelivered = d.Redelivered
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
			// RabbitMQ WILL drop channel if ACK/NACK is repeated for same delivery ID, so lock it so it can only be run once
			ev.ackLock = &sync.Mutex{}
			ev.NeedsAck = true
			ev.ack = make(chan ack)
			go func(ackCh *chan ack, delivery amqp.Delivery) {
				ackDelivery := <-*ackCh

				if ackDelivery.ack {
					delivery.Ack(true)
				} else if ackDelivery.nack {
					delivery.Nack(false, !ackDelivery.drop)
				} else {
					panic(fmt.Sprintf("%+v", ackDelivery))
				}
			}(&ev.ack, d)
		}
		ev.Body = d.Body
		c <- ev
	}
	close(c)
}

// **DESTRUCTIVE**
//
// remove all used exchanges
//
// mostly used for cleanup before tests
func (t *trMQTT) AdminCleanup() {
	ch, err := t.Conn.Channel()
	ch.ExchangeDelete("events", false, false)
	_ = err
}
func (t *trMQTT) amqpCreateEventsExchange() error {
	ch, err := t.Conn.Channel()
	if err != nil {
		return err
	}

	err = ch.ExchangeDeclare(
		t.exchange, // name
		"topic",    // type
		true,       // durable
		false,      // autoDelete
		false,      // internal
		false,      // no wait
		nil,
	)
	return err
}
