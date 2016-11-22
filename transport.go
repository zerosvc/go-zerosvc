package zerosvc

type Transport interface {
	SendEvent(path string, ev Event) error
	SendReply(path string, ev Event) error
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
	t := f(addr, cfg)
	return t
}
