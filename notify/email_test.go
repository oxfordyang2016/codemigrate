package notify

import (
	"bytes"
	"cydex"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func Test_EmailTemplateManage(t *testing.T) {
	etm := NewEmailTemplateManage("")
	Convey("Test EmailTemplateManage", t, func() {
		Convey("set base dir", func() {
			So(etm.BaseDir, ShouldEqual, "/opt/cydex/etc/ts.d/email/templates")
			etm.SetBaseDir("/tmp/email/templates")
			So(etm.BaseDir, ShouldEqual, "/tmp/email/templates")
			// empty is using default
			etm.SetBaseDir("")
			So(etm.BaseDir, ShouldEqual, "/opt/cydex/etc/ts.d/email/templates")
		})
		Convey("load lang", func() {
			etm.SetBaseDir("./../deploy/email/templates")

			var err error
			err = etm.LoadByLang("zh", false)
			So(err, ShouldBeNil)
			err = etm.LoadByLang("en", false)
			So(err, ShouldBeNil)
			err = etm.LoadByLang("en", false)
			So(err, ShouldBeNil)
		})
		Convey("render subject", func() {
			for _, v := range NotifyEvents {
				ctx := &PkgEvMailContext{
					TargetUser: &cydex.User{
						Username: "target",
					},
					PkgEvent: &PkgEvent{
						NotifyType: v,
						Time:       time.Now(),
						Pkg: &PkgInfo{
							Title:     "pkg_title",
							Notes:     "pkg_notes",
							HumanSize: "18.9K",
							NumFiles:  14,
							CreateAt:  time.Now(),
						},
						User: &cydex.User{
							Username: "user",
						},
						Owner: &cydex.User{
							Username: "owner",
						},
					},
				}
				var b bytes.Buffer
				So(b.Len(), ShouldEqual, 0)
				err := etm.RenderSubject(&b, ctx)
				So(err, ShouldBeNil)
				So(b.Len(), ShouldBeGreaterThan, 0)
			}
		})
		Convey("render content", func() {
			for _, v := range NotifyEvents {
				ctx := &PkgEvMailContext{
					TargetUser: &cydex.User{
						Username: "target",
					},
					PkgEvent: &PkgEvent{
						NotifyType: v,
						Time:       time.Now(),
						Pkg: &PkgInfo{
							Title:     "pkg_title",
							Notes:     "pkg_notes",
							HumanSize: "18.9K",
							NumFiles:  14,
							CreateAt:  time.Now(),
						},
						User: &cydex.User{
							Username: "user",
						},
						Owner: &cydex.User{
							Username: "owner",
						},
					},
				}
				var b bytes.Buffer
				So(b.Len(), ShouldEqual, 0)
				err := etm.RenderContent(&b, ctx)
				So(err, ShouldBeNil)
				So(b.Len(), ShouldBeGreaterThan, 0)
			}
		})
	})
}
