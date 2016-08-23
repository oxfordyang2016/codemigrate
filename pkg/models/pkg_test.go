package models

import (
	"./../../utils/db"
	"cydex"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
	// "time"
)

var (
	test_db = ":memory:"
	// TEST_DB  = "/tmp/job.sqlite3"
	show_sql = false
)

func Test_Pkg(t *testing.T) {
	if test_db != ":memory:" {
		os.Remove(test_db)
	}
	db.CreateEngine("sqlite3", test_db, show_sql)
	SyncTables()

	pid := "p1"
	Convey("Test Pkg", t, func() {
		Convey("add", func() {
			pkg := &Pkg{
				Pid:            pid,
				Title:          "T1",
				Notes:          "N1",
				NumFiles:       2,
				Size:           1234,
				EncryptionType: 0,
				MetaData: &cydex.MetaData{
					MtuSize: 1234,
				},
			}
			err := CreatePkg(pkg)
			So(err, ShouldBeNil)
		})
		Convey("get", func() {
			pkg, err := GetPkg(pid, false)
			So(err, ShouldBeNil)
			So(pkg, ShouldNotBeNil)
			So(pkg.Notes, ShouldEqual, "N1")
			So(pkg.MetaData, ShouldNotBeNil)
			So(pkg.MetaData.MtuSize, ShouldEqual, 1234)
		})
	})
}
