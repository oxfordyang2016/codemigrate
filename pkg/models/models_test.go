package models

import (
	"./../../db"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func Test_DBInit(t *testing.T) {
	var err error
	Convey("DB Init using sqlite3", t, func() {
		Convey("create", func() {
			err = db.CreateEngine("sqlite3", "/tmp/test.sqlite3", false)
			So(err, ShouldBeNil)
		})
		Convey("Sync Tables", func() {
			err = SyncTables()
			So(err, ShouldBeNil)
		})

	})
}
