package pkg

import (
	// "./models"
	"cydex"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func Test_DefaultUnpacker(t *testing.T) {
	uid := "1234567890ab"
	Convey("DefaultUnpacker使用服务器上的经验值", t, func() {
		U := NewDefaultUnpacker(50*1024*1024, 25)
		Convey("Test generate pid", func() {
			pid := U.GeneratePid(uid, "p1", "p1")
			So(len(pid), ShouldEqual, len(uid)+10)
		})

		Convey("Test generate fid", func() {
			pid := "1234567890ab12345678"
			var fid string
			var err error
			fid, err = U.GenerateFid(pid, 10, nil)
			So(err, ShouldBeNil)
			So(fid, ShouldEqual, "1234567890ab1234567810")
			for i := 0; i < 99; i++ {
				fid, err = U.GenerateFid(pid, i, nil)
				So(err, ShouldBeNil)
			}

			Convey("Too many files", func() {
				fid, err = U.GenerateFid(pid, 100, nil)
				So(fid, ShouldBeEmpty)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("Test generate seg", func() {
			var segs []*cydex.Seg
			var err error
			Convey("Directory", func() {
				f := &cydex.SimpleFile{
					Type: cydex.FTYPE_DIR,
					Size: 0,
				}
				segs, err = U.GenerateSegs("fid", f)
				So(segs, ShouldBeNil)
				So(err, ShouldBeNil)
			})
			Convey("Common file", func() {
				Convey("Zero size file", func() {
					f := &cydex.SimpleFile{
						Type: cydex.FTYPE_FILE,
						Size: 0,
					}
					segs, err = U.GenerateSegs("fid", f)
					So(segs, ShouldBeNil)
					So(err, ShouldBeNil)
				})
				Convey("128k", func() {
					f := &cydex.SimpleFile{
						Type: cydex.FTYPE_FILE,
						Size: 128 * 1024,
					}
					segs, err = U.GenerateSegs("ffff", f)
					So(len(segs), ShouldEqual, 1)
					So(segs[0].Size, ShouldEqual, 128*1024)
					So(segs[0].Sid, ShouldEqual, "ffff00000000")
					So(err, ShouldBeNil)
				})
				Convey("300M", func() {
					f := &cydex.SimpleFile{
						Type: cydex.FTYPE_FILE,
						Size: 300 * 1024 * 1024,
					}
					segs, err = U.GenerateSegs("ffff", f)
					So(len(segs), ShouldEqual, 6)
					So(segs[5].Size, ShouldEqual, 50*1024*1024)
					So(segs[5].Sid, ShouldEqual, "ffff00000005")
					So(err, ShouldBeNil)
				})
				Convey("1250M is (50*25)", func() {
					f := &cydex.SimpleFile{
						Type: cydex.FTYPE_FILE,
						Size: 50 * 25 * 1024 * 1024,
					}
					segs, err = U.GenerateSegs("ffff", f)
					So(len(segs), ShouldEqual, 25)
					So(segs[24].Size, ShouldEqual, 50*1024*1024)
					So(segs[24].Sid, ShouldEqual, "ffff00000024")
					So(err, ShouldBeNil)
				})
				Convey("273G 25个分片无法均分, 24个正好", func() {
					f := &cydex.SimpleFile{
						Type: cydex.FTYPE_FILE,
						Size: 273 * 1024 * 1024 * 1024,
					}
					segs, err = U.GenerateSegs("ffff", f)
					So(len(segs), ShouldEqual, 24)
					So(segs[22].Size, ShouldEqual, segs[23].Size)
					So(err, ShouldBeNil)
				})
				Convey("217G 25个分片无法均分, 会到25片", func() {
					f := &cydex.SimpleFile{
						Type: cydex.FTYPE_FILE,
						Size: 217 * 1024 * 1024 * 1024,
					}
					segs, err = U.GenerateSegs("ffff", f)
					So(len(segs), ShouldEqual, 25)
					So(segs[22].Size, ShouldEqual, segs[23].Size)
					So(segs[24].Size, ShouldBeLessThan, segs[23].Size)
					So(err, ShouldBeNil)
				})
			})
		})
	})
}
