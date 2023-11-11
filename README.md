# go-zeromq
[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/zerosvc/go-zerosvc)

Go bindings for [zerosvc](https://github.com/zerosvc/zerosvc)




## Event field mapping

MQTTv3 format is whole event serialized into JSON
AMQP/MQTTv5 uses headers and puts body as-is

| event | MQTTv3 | MQTTv5 (future) | AMQP |
| ---   |  ----  |  ---  | --- |
|  Reply to  | body json    | UserProperty[reply_to]      | Header[reply_to]  |
| Redelivered | DUP transport flag| DUP transport flag | redelivered flag | 
| RetainTill  | retain flag (**NO TTL**)| retain + Expiry | no support |
| Headers | body JSON | UserProperty | Headers[] |
| Body | body JSON['body'] | Body | Body |

+ extra transport control headers:

| header key       | MQTTv3 | MQTTv5   | AMQP   |
| ---              | ---    | ---      |  ---   |
| `_transport_ttl` | ---    | Expiry   | TTL    | 
| `node-name`      | ---    | retained | AppId  |
| `correlation-id` | ---    | retained | CorrelationId  |
| `user-id`        | ---    | retained | UserId  |

other:

* topic of incoming messages is encoded in `RoutingKey


## MQTT URL options

* username/pass - same as basic auth e.g. `tcp://mqttuser:mqttpass@127.0.0.1:1883`
* TLS - just change proto to `tls` - `tls://mqtt.example.com:8883`
* query options
  * CA - `tls://mqtt.example.com:8883/?ca=/my/ca/path.crt`
  * client cert (key + cert in one file) - `tls://mqtt.example.com:8883/?cert=/path/certandkey.pem`
  * client cert (key, cert in separate file) - `tls://mqtt.example.com:8883/?cert=/path/cert.pem&certkey=/path/certkey.pem`



## Examples

### Create node

```go
tr := zerosvc.NewTransport(zerosvc.TransportMQTT,"tcp://127.0.0.1:1883",zerosvc.TransportMQTTConfig{})
	node := zerosvc.NewNode("unique-node-name")
	tr.Connect()
	node.SetTransport(tr)
```

prepare and send event. `Prepare()` takes care of checksums/signatures for the event

```
	e := zerosvc.Event{
		Headers:    map[string]interface{}{
			"test": "header",
		},
		Body:        []byte("hay"),
	}
	e.Prepare()
	node.SendEvent("dpp/ev",e)
```

## Quirks

* Due to how MQTT libraries work only first user/password is used for all urls.