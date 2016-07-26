package models

import (
	"./../../db"
	"cydex"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
	"time"
)

var (
	TEST_DB = ":memory:"
	// TEST_DB  = "/tmp/job.sqlite3"
	SHOW_SQL = false
)

func Test_Job(t *testing.T) {
	if TEST_DB != ":memory:" {
		os.Remove(TEST_DB)
	}
	db.CreateDefaultDBEngine("sqlite3", TEST_DB, SHOW_SQL)
	DBSyncTables()

	Convey("Test Job", t, func() {
		Convey("Test create", func() {
			j, err := CreateJob("123", "1", "2", cydex.UPLOAD)
			So(err, ShouldBeNil)
			So(j.Finished, ShouldBeFalse)
		})
		Convey("Test update", func() {
			j, err := CreateJob("321", "1", "2", cydex.UPLOAD)
			So(err, ShouldBeNil)
			j.Finish()
			So(j.Finished, ShouldBeTrue)
		})
	})
}

func Test_JobDetail(t *testing.T) {
	Convey("Test JobDetail", t, func() {
		Convey("Test create", func() {
			jd, err := CreateJobDetail("123", "4567")
			So(err, ShouldBeNil)
			So(jd.FinishedSize, ShouldEqual, 0)
		})
		Convey("Test update", func() {
			jd, err := CreateJobDetail("321", "4567")
			So(err, ShouldBeNil)
			So(jd.UpdateAt.IsZero(), ShouldBeFalse)
			jd.Finish()
			So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_DONE)
		})
		Convey("Test Save", func() {
			jd, err := CreateJobDetail("111", "4567")
			So(err, ShouldBeNil)
			So(jd.UpdateAt.Sub(jd.CreateAt), ShouldBeLessThan, 1*time.Second)
			err = jd.SetStartTime(time.Time{})
			So(err, ShouldBeNil)
			time.Sleep(time.Second)
			err = jd.Save()
			So(jd.UpdateAt.Sub(jd.CreateAt), ShouldBeGreaterThanOrEqualTo, 1*time.Second)
		})
	})
}
