package models

import (
	"./../../db"
	"cydex"
	"fmt"
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
	db.CreateEngine("sqlite3", TEST_DB, SHOW_SQL)
	SyncTables()

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

func Test_DeleteJob(t *testing.T) {
	j, _ := CreateJob("aaa", "u1", "p1", cydex.DOWNLOAD)
	for i := 0; i < 2; i++ {
		CreateJobDetail(j.JobId, fmt.Sprintf("fid_%d", i))
	}
	j, _ = CreateJob("bbb", "u2", "p2", cydex.DOWNLOAD)
	for i := 0; i < 3; i++ {
		CreateJobDetail(j.JobId, fmt.Sprintf("fid_%d", i))
	}

	Convey("test delete", t, func() {
		DeleteJob("aaa")
		j, _ := GetJob("aaa", false)
		So(j, ShouldBeNil)
		j, _ = GetJob("bbb", false)
		So(j, ShouldNotBeNil)

		jds, _ := GetJobDetails("bbb")
		So(jds, ShouldHaveLength, 3)

		jds, _ = GetJobDetails("aaa")
		So(jds, ShouldHaveLength, 0)
	})
}
