package zerosvc

import (
	"fmt"
)

func ExampleEvent_Marshal() {
	node := NewNode("testnode", "77ab2b23-4f1b-4247-be45-dcc2d93ffb94")
	ev := node.NewEvent()
	out := make(map[string]int)
	out["test"] = 4
	ev.Marshal(out)
	fmt.Printf("-- %s --", string(ev.Body))
	// Output: -- {"test":4} --
}

func ExampleEvent_Prepare() {
	node := NewNode("testnode", "77ab2b23-4f1b-4247-be45-dcc2d93ffb94")
	ev := node.NewEvent()
	ev.Body = []byte("test")
	ev.Prepare()
	fmt.Printf("-- %s --", string(ev.Headers["sha256"].(string)))
	// Output: -- 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08 --
}

func ExampleEvent_Unmarshal() {
	node := NewNode("testnode", "77ab2b23-4f1b-4247-be45-dcc2d93ffb94")
	ev := node.NewEvent()
	ev.Body = []byte(`{"test":6}`)
	out := make(map[string]int)
	ev.Unmarshal(&out)
	fmt.Printf("-- %d --", out["test"])
	// Output: -- 6 --
}
