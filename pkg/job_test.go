package pkg

import (
	"./../db"
	"./../transfer/task"
	"./models"
	"cydex"
	"cydex/transfer"
	// "fmt"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
)

const (
	// TEST_DB = ":memory:"
	TEST_DB = "/tmp/job.sqlite3"
)

func initDB() {
	if TEST_DB != ":memory" {
		os.Remove(TEST_DB)
	}
	db.CreateEngine("sqlite3", TEST_DB, false)
	models.SyncTables()
}

func Test_CreateJob(t *testing.T) {
	var err error
	pid := "1234567890ab1111122222"
	fid1 := "1234567890111112222201"
	sid1_of_fid1 := "123456789011111222220100000001"
	fid2 := "1234567890111112222202"
	sid1_of_fid2 := "123456789011111222220200000001"

	initDB()
	JobMgr.SetCacheSyncTimeout(0)

	Convey("Test CreateJob", t, func() {
		Convey("Create pkg records first", func() {
			_, err = models.CreatePkg(pid, "test", "test", 2, 5000, cydex.ENCRYPTION_TYPE_AES256)
			So(err, ShouldBeNil)

			_, err = models.CreateFile(fid1, pid, "1.txt", "/tmp", 2000, 1)
			So(err, ShouldBeNil)

			_, err = models.CreateSeg(sid1_of_fid1, fid1, 2000)
			So(err, ShouldBeNil)

			_, err = models.CreateFile(fid2, pid, "2.txt", "/tmp", 3000, 1)
			So(err, ShouldBeNil)

			_, err = models.CreateSeg(sid1_of_fid2, fid2, 3000)
			So(err, ShouldBeNil)
		})

		Convey("Create upload job", func() {
			_, err = JobMgr.CreateJob("1234567890ab", pid, cydex.UPLOAD)
			So(err, ShouldBeNil)
			hashid := GenerateHashId("1234567890ab", pid, cydex.UPLOAD)
			j, err := models.GetJob(hashid, true)
			So(err, ShouldBeNil)
			So(j, ShouldNotBeNil)
			j = JobMgr.getJob(hashid)
			So(j, ShouldNotBeNil)
		})

		Convey("Create download job", func() {
			_, err = JobMgr.CreateJob("ab1234567890", pid, cydex.DOWNLOAD)
			So(err, ShouldBeNil)
			hashid := GenerateHashId("ab1234567890", pid, cydex.DOWNLOAD)
			j, err := models.GetJob(hashid, true)
			So(err, ShouldBeNil)
			So(j, ShouldNotBeNil)
			j = JobMgr.getJob(hashid)
			So(j, ShouldNotBeNil)

			hashid = GenerateHashId("kk1234567890", pid, cydex.DOWNLOAD)
			j, err = models.GetJob(hashid, true)
			So(j, ShouldBeNil)
		})

		Convey("update task", func() {
			hashid := GenerateHashId("1234567890ab", pid, cydex.UPLOAD)
			j := JobMgr.getJob(hashid)

			Convey("add task", func() {
				jd := JobMgr.getJobDetail(j, fid1)
				So(jd.StartTime.IsZero(), ShouldBeTrue)
				t := &task.Task{
					TaskId: "t1",
					Type:   cydex.UPLOAD,
					UploadReq: &task.UploadReq{
						UploadTaskReq: &transfer.UploadTaskReq{
							Uid:     "1234567890ab",
							Fid:     fid1,
							SidList: []string{sid1_of_fid1},
							Size:    2000,
						},
						Pid: pid,
					},
				}
				JobMgr.AddTask(t)
				So(jd.StartTime.IsZero(), ShouldBeFalse)
			})

			Convey("task transferring", func() {
				jd := JobMgr.getJobDetail(j, fid1)
				So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_NONE)
				state := &transfer.TaskState{
					TaskId:     "t1",
					Sid:        sid1_of_fid1,
					State:      "transferring",
					TotalBytes: 1234,
					Bitrate:    123,
				}
				t := &task.Task{
					TaskId: "t1",
					Type:   cydex.UPLOAD,
					UploadReq: &task.UploadReq{
						UploadTaskReq: &transfer.UploadTaskReq{
							Uid:     "1234567890ab",
							Fid:     fid1,
							SidList: []string{sid1_of_fid1},
							Size:    2000,
						},
						Pid: pid,
					},
				}
				JobMgr.TaskStateNotify(t, state)
				So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_DOING)
			})

			Convey("task end", func() {
				jd := JobMgr.getJobDetail(j, fid1)
				So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_DOING)
				state := &transfer.TaskState{
					TaskId:     "t1",
					Sid:        sid1_of_fid1,
					State:      "end",
					TotalBytes: 2000,
					Bitrate:    123,
				}
				t := &task.Task{
					TaskId: "t1",
					Type:   cydex.UPLOAD,
					UploadReq: &task.UploadReq{
						UploadTaskReq: &transfer.UploadTaskReq{
							Uid:     "1234567890ab",
							Fid:     fid1,
							SidList: []string{sid1_of_fid1},
							Size:    2000,
						},
						Pid: pid,
					},
				}
				JobMgr.TaskStateNotify(t, state)
				So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_DONE)

				So(j.NumFinishedDetails, ShouldEqual, 1)
			})

			Convey("job end", func() {
				So(j.Finished, ShouldBeFalse)

				jd := JobMgr.getJobDetail(j, fid2)
				state := &transfer.TaskState{
					TaskId:     "t2",
					Sid:        sid1_of_fid2,
					State:      "end",
					TotalBytes: 3000,
					Bitrate:    123,
				}
				t := &task.Task{
					TaskId: "t2",
					Type:   cydex.UPLOAD,
					UploadReq: &task.UploadReq{
						UploadTaskReq: &transfer.UploadTaskReq{
							Uid:     "1234567890ab",
							Fid:     fid2,
							SidList: []string{sid1_of_fid2},
							Size:    3000,
						},
						Pid: pid,
					},
				}
				JobMgr.TaskStateNotify(t, state)
				So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_DONE)

				So(j.NumFinishedDetails, ShouldEqual, 2)
				So(j.Finished, ShouldBeTrue)
				So(JobMgr.HasCachedJob(j.JobId), ShouldBeFalse)
			})
		})
	})
}
