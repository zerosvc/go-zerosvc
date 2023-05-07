package zerosvc

type Message struct {
	Topic   string
	Payload []byte
}

type Hooks struct {
	ConnectHook        func()
	ConnectionLossHook func(err error)
}
