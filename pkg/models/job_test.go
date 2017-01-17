package models

import (
	"./../../utils/db"
	"cydex"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
	"time"
)

var (
	// TEST_DB = ":memory:"
	TEST_DB  = "/tmp/job.sqlite3"
	SHOW_SQL = true
)

func init() {
	resetDB()
}

func resetDB() {
	if TEST_DB != ":memory:" {
		os.Remove(TEST_DB)
	}
	db.CreateEngine("sqlite3", TEST_DB, SHOW_SQL)
	SyncTables()
}

func Test_Job(t *testing.T) {
	Convey("Test Job", t, func() {
		Convey("create", func() {
			_, err := CreateJob("123", "u1", "p1", cydex.UPLOAD)
			So(err, ShouldBeNil)
			// So(j.State, ShouldEqual, cydex.TRANSFER_STATE_IDLE)
		})
		// Convey("Test update", func() {
		// 	j, err := CreateJob("321", "1", "2", cydex.DOWNLOAD)
		// 	So(err, ShouldBeNil)
		//
		// 	j.SetState(cydex.TRANSFER_STATE_DOING)
		// 	So(j.State, ShouldEqual, cydex.TRANSFER_STATE_DOING)
		// 	j, err = GetJob("321", false)
		// 	So(j.State, ShouldEqual, cydex.TRANSFER_STATE_DOING)
		// 	So(j.FinishAt.IsZero(), ShouldBeTrue)
		//
		// 	j.SetState(cydex.TRANSFER_STATE_DONE)
		// 	So(j.State, ShouldEqual, cydex.TRANSFER_STATE_DONE)
		// 	j, err = GetJob("321", false)
		// 	So(j.State, ShouldEqual, cydex.TRANSFER_STATE_DONE)
		// 	So(j.FinishAt.IsZero(), ShouldBeFalse)
		//
		// 	// 不影响其他记录
		// 	j, err = GetJob("123", false)
		// 	So(j.State, ShouldEqual, cydex.TRANSFER_STATE_IDLE)
		// })
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
		Convey("CountUnfinished", func() {
			j, _ := CreateJob("kkk", "u1", "p1", cydex.DOWNLOAD)
			for i := 0; i < 10; i++ {
				jd, _ := CreateJobDetail(j.JobId, fmt.Sprintf("fid_%d", i))
				if i%2 == 0 {
					jd.SetState(cydex.TRANSFER_STATE_DONE)
				}
			}
			n, err := CountUnfinishedJobDetails("kkk")
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 5)

			// job_id not existed
			n, err = CountUnfinishedJobDetails("yyy")
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 0)
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

func Test_JobSearch(t *testing.T) {
	resetDB()

	times := []string{
		"2016-12-17 17:12:30",
		"2016-04-17 19:03:59",
		"2017-01-05 22:43:12",
	}
	pkgs := []*Pkg{
		&Pkg{Pid: "p1", Title: "test", Size: 876543, NumFiles: 5},
		&Pkg{Pid: "p2", Title: "中文测试", Size: 1401357914234, NumFiles: 18},
		&Pkg{Pid: "p3", Title: "333", Size: 1368364, NumFiles: 99},
	}
	jobs := []*Job{
		&Job{JobId: "job1", Uid: "u1", Pid: "p1", Type: cydex.UPLOAD},
		&Job{JobId: "job2", Uid: "u1", Pid: "p2", Type: cydex.UPLOAD},
		&Job{JobId: "job3", Uid: "u2", Pid: "p3", Type: cydex.UPLOAD},
	}
	for i, pkg := range pkgs {
		CreatePkg(pkg)
		// 更新create_at, xorm默认使用当前时间插入
		sql := "update package_pkg set create_at=? where id=?"
		DB().Exec(sql, times[i], pkg.Id)
	}
	for i, job := range jobs {
		DB().Insert(job)
		sql := "update package_job set create_at=? where id=?"
		DB().Exec(sql, times[i], job.Id)
	}

	Convey("job search", t, func() {
		Convey("filter title", func() {
			filter := JobFilter{
				Title: "文",
			}
			jobs, err := GetJobsEx(cydex.UPLOAD, nil, &filter)
			So(err, ShouldBeNil)
			So(jobs, ShouldHaveLength, 1)
			So(jobs[0].Pid, ShouldEqual, "p2")
		})
		Convey("filter sort by create_at", func() {
			{
				filter := JobFilter{
					OrderBy: "create",
				}
				jobs, err := GetJobsEx(cydex.UPLOAD, nil, &filter)
				So(err, ShouldBeNil)
				So(jobs, ShouldHaveLength, 3)
				So(jobs[0].Pid, ShouldEqual, "p2")
				So(jobs[2].Pid, ShouldEqual, "p3")
			}

			{
				filter := JobFilter{
					OrderBy: "-create",
				}
				jobs, err := GetJobsEx(cydex.UPLOAD, nil, &filter)
				So(err, ShouldBeNil)
				So(jobs, ShouldHaveLength, 3)
				So(jobs[0].Pid, ShouldEqual, "p3")
				So(jobs[2].Pid, ShouldEqual, "p2")
			}
		})
		Convey("filter sort by size", func() {
			{
				filter := JobFilter{
					OrderBy: "size",
				}
				jobs, err := GetJobsEx(cydex.UPLOAD, nil, &filter)
				So(err, ShouldBeNil)
				So(jobs, ShouldHaveLength, 3)
				So(jobs[0].Pid, ShouldEqual, "p1")
				So(jobs[2].Pid, ShouldEqual, "p2")
			}

			{
				filter := JobFilter{
					OrderBy: "-size",
				}
				jobs, err := GetJobsEx(cydex.UPLOAD, nil, &filter)
				So(err, ShouldBeNil)
				So(jobs, ShouldHaveLength, 3)
				So(jobs[0].Pid, ShouldEqual, "p2")
				So(jobs[2].Pid, ShouldEqual, "p1")
			}
		})
		Convey("filter by owner", func() {
			filter := JobFilter{
				Owner: "u1",
			}
			jobs, err := GetJobsEx(cydex.UPLOAD, nil, &filter)
			So(err, ShouldBeNil)
			So(jobs, ShouldHaveLength, 2)
			So(jobs[0].Pid, ShouldEqual, "p1")
			So(jobs[1].Pid, ShouldEqual, "p2")
		})
		Convey("filter by datetime", func() {
			filter := JobFilter{
				BegTime: time.Date(2016, time.June, 12, 0, 0, 0, 0, time.UTC),
				EndTime: time.Date(2016, time.December, 30, 23, 59, 59, 0, time.UTC),
			}
			jobs, err := GetJobsEx(cydex.UPLOAD, nil, &filter)
			So(err, ShouldBeNil)
			So(jobs, ShouldHaveLength, 1)
			So(jobs[0].Pid, ShouldEqual, "p1")
		})
		Convey("no filter", func() {
			jobs, err := GetJobsEx(cydex.UPLOAD, nil, nil)
			So(err, ShouldBeNil)
			So(jobs, ShouldHaveLength, 3)
			So(jobs[0].Pid, ShouldEqual, "p3")
			So(jobs[2].Pid, ShouldEqual, "p2")
		})
	})
}
