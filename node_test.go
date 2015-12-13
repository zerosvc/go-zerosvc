package zerosvc

import (
	//	"bufio"
	//	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	//	"os"
	//	"strings"
	"testing"
)

func TestNode(t *testing.T) {
	node1 := NewNode("testnode1", "77ab2b23-4f1b-4247-be45-dcc2d93ffb9c")
	Convey("New node", t, func() {
		So(node1.Name, ShouldEqual, "testnode1")
		So(node1.UUID, ShouldEqual, "77ab2b23-4f1b-4247-be45-dcc2d93ffb9c")
	})
	node2 := NewNode("testnode2")
	Convey("New node without UUID", t, func() {
		So(node2.Name, ShouldEqual, "testnode2")
		So(node2.UUID, ShouldNotEqual, "")
	})
	Convey("New node UUID should be deterministic", t, func() {
		node3 := NewNode("testnode3")
		node4 := NewNode("testnode3")
		So(node3.UUID, ShouldEqual, node4.UUID)
	})
}
