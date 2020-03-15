# go-zeromq
[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/zerosvc/go-zerosvc)

Go bindings for [zerosvc](https://github.com/zerosvc/zerosvc)


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