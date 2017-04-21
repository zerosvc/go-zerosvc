package zerosvc

type Transport interface {
	SendEvent(path string, ev Event) error
	SendReply(path string, ev Event) error
	// returns channel with new events. channel will close on error
	GetEvents(filter string, channel chan Event) error
	Connect() error
	// Shutdown should be called at the end of app
	Shutdown()
	AdminCleanup()
}

func NewTransport(f func(string, interface{}) Transport, addr string, cfg ...interface{}) Transport {
	if len(cfg) < 1 {
		cfg = make([]interface{}, 1)
	}
	if len(cfg) > 0 {
		return f(addr, cfg[0])
	} else {
		return f(addr, nil)
	}
}
