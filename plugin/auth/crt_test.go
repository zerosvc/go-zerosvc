package auth

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestDummy(t *testing.T) {
	a := New()
	err := a.LoadCA("../../t-data/ca-crt.pem")
	Convey("Load cert", t, func() {
		So(err, ShouldEqual, nil)
	})
}
