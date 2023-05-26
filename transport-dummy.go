package zerosvc

import (
	"testing"
)

type trDummy struct {
	Transport
	T *testing.T
}

// Dummy/debug transport
func TransportDummy(addr string, cfg interface{}) Transport {
	var t trDummy
	return &t
}

// print msg to stdout
func (t *trDummy) SendEvent(path string, ev Event) error {
	if t.T != nil {
		t.T.Logf("SendEvent path: %s, data: %+v\n", path, ev)
	}
	var err error
	return err
}

func (t *trDummy) GetEvents(path string, ch chan Event) error {
	var err error
	node := NewNode("testnode", "77ab2b23-4f1b-4247-be45-dcc2d93ffb94")
	ev := node.NewEvent()
	ev.Headers["path"] = path
	ev.Body = []byte("dummy event")
	ev.Prepare()
	ch <- ev
	return err
}

func (t *trDummy) Connect() error {
	var err error
	if t.T != nil {
		t.T.Logf("Put your connection start here\n")
	}
	return err
}
func (t *trDummy) SetupHeartbeat(path string) {

}
