package zerosvc

type Message struct {
	Topic           string
	ResponseTopic   string
	CorrelationData []byte
	ContentType     string
	Metadata        map[string]string
	Payload         []byte
	Retain          bool
}

type Hooks struct {
	ConnectHook        func()
	ConnectionLossHook func(err error)
}
