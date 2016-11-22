package zerosvc

import (
	"github.com/streadway/amqp"
	"os"
	"time"
)


type trAMQP struct {
	Transport
	addr string
	Conn *amqp.Connection
}

func TransportAMQP(addr string, cfg interface{}) Transport {
	var t *trAMQP = &trAMQP{}
	t.addr = addr
	return t
}

func (t *trAMQP) Connect() error {
	var err error
	conn, err := amqp.Dial(t.addr)
	if err != nil {
		return err
	}
	t.Conn = conn
	return err
}
func (t *trAMQP) Shutdown() {
	t.Conn.Close()
}

func (t *trAMQP) SendEvent(path string, ev Event) error {
	ch, err := t.Conn.Channel()
	if err != nil {
		return err
	}
	//
	//	headers := make(amqp.Table)
	//	headers["version"] = "1.2.2"

	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body:         ev.Body,
	}
	// map headers to AMQP ones
	has := func(key string) bool { _, ok := ev.Headers[key]; return ok }
	if has("node-name") {
		msg.AppId = ev.Headers["node-name"].(string)
		delete(ev.Headers,"node-name")
	}
	if has("correlation-id") {
		msg.CorrelationId = ev.Headers["correlation-id"].(string)
		delete(ev.Headers,"correlation-id")
	}
	msg.Headers =  ev.Headers


	err = ch.Publish("events", path, false, false, msg)
	return err
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
	q, err := amqpCreateAndBindQueue(ch, filter)
	// we only try to create exchange to not take the cost on checking if exchange exists on every connection
	if err != nil {
		err = t.amqpCreateEventsExchange()
		if err != nil {
			return err
		} else {
			ch, err = t.Conn.Channel()
			q, err = amqpCreateAndBindQueue(ch, filter)
			if err != nil {
				return err
			}
		}
	}
	go amqpEventReceiver(ch, q, channel)
	return err
}

func amqpCreateAndBindQueue(ch *amqp.Channel, filter string) (amqp.Queue, error) {
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
		"events", // exchange
		false,
		nil,
	)
	if err != nil {
		return q, err
	}
	return q, err
}

func amqpEventReceiver(ch *amqp.Channel, q amqp.Queue, c chan Event) {
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
		"events", // name
		"topic",  // type
		true,     // durable
		false,    // autoDelete
		false,    // internal
		false,    // no wait
		nil,
	)
	return err
}
