package zerosvc

import (
	//	"bufio"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	//	"os"
	//	"strings"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestEvent(t *testing.T) {
	node := NewNode("testnode", "77ab2b23-4f1b-4247-be45-dcc2d93ffb94")
	ev := node.NewEvent()
	Convey("Passed init", t, func() {
		So(ev.Headers[`node-name`], ShouldEqual, "testnode")
		So(ev.Headers[`node-uuid`], ShouldEqual, "77ab2b23-4f1b-4247-be45-dcc2d93ffb94")
		So(ev.Headers[`ts`], ShouldResemble, time.Unix(0, 0))
		So(ev.Headers[`sha256`], ShouldResemble, "")
	})
	Convey("Prepared event", t, func() {

		type Bo struct {
			CakeCount int    `json:"cake_count"`
			CakeType  string `json:"cake_type"`
		}
		bo := Bo{
			CakeCount: 10,
			CakeType:  "Chocolate",
		}

		err := ev.Marshal(bo)
		Convey("Serializing successful", func() {
			So(err, ShouldEqual, nil)
			So(string(ev.Body), ShouldNotEqual, "{}")
		})
		body := ev.Body
		body256 := sha256.Sum256(body)
		err = ev.Prepare()
		So(err, ShouldEqual, nil)
		Convey("ts bigger than 0", func() {
			So(ev.Headers[`ts`], ShouldNotResemble, time.Unix(0, 0))
		})
		Convey("sha256 sum correct", func() {
			So(ev.Headers[`sha256`], ShouldResemble, (hex.EncodeToString(body256[:32])))
		})
		Convey("Unmarshal works", func() {
			var un Bo = Bo{}
			err := ev.Unmarshal(&un)
			So(err, ShouldEqual, nil)
			So(fmt.Sprintf("%+v", un), ShouldResemble, fmt.Sprintf("%+v", bo))
		})
	})
}

func BenchmarkNewEvent(b *testing.B) {
	node := NewNode("testnode", "77ab2b23-4f1b-4247-be45-dcc2d93ffb94")

	for i := 0; i < b.N; i++ {
		ev := node.NewEvent()
		ev.Body = []byte("test")
		ev.Prepare()
	}
}
