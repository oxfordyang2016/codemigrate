package pkg

import (
	trans_model "./../transfer/models"
	"./../transfer/task"
	"./../utils/db"
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

func init() {
	initDB()
	JobMgr.SetCacheSyncTimeout(0)
	JobMgr.del_job_delay = 0
}

func initDB() {
	if TEST_DB != ":memory" {
		os.Remove(TEST_DB)
	}
	db.CreateEngine("sqlite3", TEST_DB, false)
	models.SyncTables()
}

func Test_Track(t *testing.T) {
	u1 := "u1"
	p1 := "p1"
	u2 := "u2"
	u3 := "u3"
	p3 := "p3"

	Convey("Test track", t, func() {
		Convey("Add track", func() {
			JobMgr.AddTrack(u1, p1, cydex.UPLOAD, true)
			JobMgr.AddTrack(u3, p3, cydex.UPLOAD, true)

			JobMgr.AddTrack(u2, p1, cydex.DOWNLOAD, true)
			JobMgr.AddTrack(u2, p3, cydex.DOWNLOAD, true)
			JobMgr.AddTrack(u3, p1, cydex.DOWNLOAD, true)

			So(JobMgr.track_users, ShouldHaveLength, 3)
			So(JobMgr.track_pkgs, ShouldHaveLength, 2)
		})

		Convey("get user track", func() {
			pids := JobMgr.GetUserTrack(u2, cydex.DOWNLOAD)
			So(pids, ShouldHaveLength, 2)
			So(p3, ShouldBeIn, pids)
			So(p1, ShouldBeIn, pids)

			pids = JobMgr.GetUserTrack(u1, cydex.DOWNLOAD)
			So(pids, ShouldBeEmpty)

			pids = JobMgr.GetUserTrack(u3, cydex.DOWNLOAD)
			So(pids, ShouldHaveLength, 1)
			So(p1, ShouldBeIn, pids)
		})

		Convey("get pkg track", func() {
			uids := JobMgr.GetPkgTrack(p1, cydex.DOWNLOAD)
			So(uids, ShouldHaveLength, 2)
			So(u2, ShouldBeIn, uids)
			So(u3, ShouldBeIn, uids)
			So(u1, ShouldNotBeIn, uids)

			uids = JobMgr.GetPkgTrack(p3, cydex.DOWNLOAD)
			So(uids, ShouldHaveLength, 1)
			So(u2, ShouldBeIn, uids)
		})

		Convey("del track", func() {
			uids := JobMgr.GetPkgTrack(p1, cydex.UPLOAD)
			So(uids, ShouldHaveLength, 1)

			JobMgr.DelTrack(u1, p1, cydex.UPLOAD, true)
			uids = JobMgr.GetPkgTrack(p1, cydex.UPLOAD)
			// 没下载完, 上传track不删档
			So(uids, ShouldHaveLength, 1)

			JobMgr.DelTrack(u2, p1, cydex.DOWNLOAD, true)
			JobMgr.DelTrack(u3, p1, cydex.DOWNLOAD, true)
			uids = JobMgr.GetPkgTrack(p1, cydex.UPLOAD)
			So(uids, ShouldHaveLength, 0)

			// nothing happend if uid or pid is not match
			JobMgr.DelTrack("no this uid", p1, cydex.UPLOAD, true)

			// nothing happend to p3
			uids = JobMgr.GetPkgTrack(p3, cydex.DOWNLOAD)
			So(uids, ShouldHaveLength, 1)
			So(u2, ShouldBeIn, uids)
		})
	})
}

type FakeJobObserver struct {
	upload_job_create_v   int
	download_job_create_v int
	finished_v            int
	start_job             *models.Job
	finish_job            *models.Job
}

func (self *FakeJobObserver) OnJobCreate(job *models.Job) {
	if job.Type == cydex.UPLOAD {
		self.upload_job_create_v += 1
	} else {
		self.download_job_create_v += 1
	}
}

func (self *FakeJobObserver) OnJobStart(job *models.Job) {
	self.start_job = job
}

func (self *FakeJobObserver) OnJobFinish(job *models.Job) {
	self.finish_job = job
}

func Test_CreateJob(t *testing.T) {
	var err error
	pid := "1234567890ab1111122222"
	fid1 := "1234567890111112222201"
	sid1_of_fid1 := "123456789011111222220100000001"
	fid2 := "1234567890111112222202"
	sid1_of_fid2 := "123456789011111222220200000001"

	fake_job_observer := new(FakeJobObserver)
	JobMgr.AddJobObserver(fake_job_observer)

	Convey("Test CreateJob", t, func() {
		Convey("Create pkg records first", func() {
			pkg := &models.Pkg{
				Pid:            pid,
				Title:          "test",
				Notes:          "test",
				NumFiles:       2,
				Size:           5000,
				EncryptionType: cydex.ENCRYPTION_TYPE_AES256,
				MetaData: &cydex.MetaData{
					MtuSize: 1234,
				},
			}
			err = models.CreatePkg(pkg)
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
			err = JobMgr.CreateJob("1234567890ab", pid, cydex.UPLOAD)
			So(err, ShouldBeNil)
			hashid := HashJob("1234567890ab", pid, cydex.UPLOAD)
			j, err := models.GetJob(hashid, true)
			So(err, ShouldBeNil)
			So(j, ShouldNotBeNil)
			j = JobMgr.GetJob(hashid)
			So(j, ShouldNotBeNil)
			So(fake_job_observer.upload_job_create_v, ShouldEqual, 1)
		})

		Convey("Create download job", func() {
			err = JobMgr.CreateJob("ab1234567890", pid, cydex.DOWNLOAD)
			So(err, ShouldBeNil)
			hashid := HashJob("ab1234567890", pid, cydex.DOWNLOAD)
			j, err := models.GetJob(hashid, true)
			So(err, ShouldBeNil)
			So(j, ShouldNotBeNil)
			j = JobMgr.GetJob(hashid)
			So(j, ShouldNotBeNil)
			So(fake_job_observer.download_job_create_v, ShouldEqual, 1)

			hashid = HashJob("not_existed_uid", pid, cydex.DOWNLOAD)
			j, err = models.GetJob(hashid, true)
			So(j, ShouldBeNil)
		})

		Convey("update task", func() {
			hashid := HashJob("1234567890ab", pid, cydex.UPLOAD)
			j := JobMgr.GetJob(hashid)

			Convey("check first", func() {
				So(j.IsCached, ShouldBeTrue)
				So(j.NumUnfinishedDetails, ShouldEqual, 2)
			})
			Convey("add task", func() {
				jd := JobMgr.GetJobDetail(j.JobId, fid1)
				So(jd.StartTime.IsZero(), ShouldBeTrue)
				t := &task.Task{
					Task: &trans_model.Task{
						TaskId: "t1",
						JobId:  hashid,
						Fid:    fid1,
						Type:   cydex.UPLOAD,
					},
				}
				JobMgr.AddTask(t)
				So(jd.StartTime.IsZero(), ShouldBeFalse)

				So(fake_job_observer.start_job.Pid, ShouldEqual, pid)
				So(fake_job_observer.start_job.Uid, ShouldEqual, "1234567890ab")
			})

			Convey("task transferring", func() {
				jd := JobMgr.GetJobDetail(j.JobId, fid1)
				So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_IDLE)
				state := &transfer.TaskState{
					TaskId:     "t1",
					Sid:        sid1_of_fid1,
					State:      "transferring",
					TotalBytes: 1234,
					Bitrate:    123,
				}
				t := &task.Task{
					Task: &trans_model.Task{
						TaskId: "t1",
						JobId:  hashid,
						Fid:    fid1,
						Type:   cydex.UPLOAD,
						State:  cydex.TRANSFER_STATE_DOING,
					},
				}
				JobMgr.TaskStateNotify(t, state)
				So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_DOING)
				So(jd.NumFinishedSegs, ShouldEqual, 0)
				So(jd.FinishedSize, ShouldEqual, 0)
			})

			Convey("task sid end", func() {
				jd := JobMgr.GetJobDetail(j.JobId, fid1)
				So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_DOING)
				state := &transfer.TaskState{
					TaskId:     "t1",
					Sid:        sid1_of_fid1,
					State:      "end",
					TotalBytes: 2000,
					Bitrate:    123,
				}
				t := &task.Task{
					Task: &trans_model.Task{
						TaskId: "t1",
						JobId:  hashid,
						Fid:    fid1,
						Type:   cydex.UPLOAD,
						State:  cydex.TRANSFER_STATE_DONE,
					},
				}
				JobMgr.TaskStateNotify(t, state)
				So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_DONE)

				So(j.NumUnfinishedDetails, ShouldEqual, 1)
			})

			Convey("job end", func() {
				So(j.IsFinished(), ShouldBeFalse)

				So(fake_job_observer.finish_job, ShouldBeNil)

				jd := JobMgr.GetJobDetail(j.JobId, fid2)
				state := &transfer.TaskState{
					TaskId:     "t2",
					Sid:        sid1_of_fid2,
					State:      "end",
					TotalBytes: 3000,
					Bitrate:    123,
				}
				t := &task.Task{
					Task: &trans_model.Task{
						TaskId: "t2",
						JobId:  hashid,
						Fid:    fid2,
						Type:   cydex.UPLOAD,
						State:  cydex.TRANSFER_STATE_DONE,
					},
				}
				JobMgr.TaskStateNotify(t, state)
				So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_DONE)

				// So(j.NumFinishedDetails, ShouldEqual, 2)
				So(j.NumUnfinishedDetails, ShouldEqual, 0)
				So(j.IsFinished(), ShouldBeTrue)
				So(JobMgr.HasCachedJob(j.JobId), ShouldBeFalse)

				So(fake_job_observer.finish_job, ShouldEqual, j)
			})

			Convey("task self interrupted", func() {
				jd := JobMgr.GetJobDetail(j.JobId, fid1)
				So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_DONE)
				t := &task.Task{
					Task: &trans_model.Task{
						TaskId: "t1",
						JobId:  hashid,
						Fid:    fid1,
						Type:   cydex.UPLOAD,
						State:  cydex.TRANSFER_STATE_PAUSE,
					},
				}
				JobMgr.DelTask(t)
				jd = JobMgr.GetJobDetail(j.JobId, fid1)
				So(jd.State, ShouldEqual, cydex.TRANSFER_STATE_PAUSE)
			})
		})
	})
}

func Test_TrackOfDelete(t *testing.T) {
	u1 := "u1"
	p1 := "p1"
	u2 := "u2"
	p2 := "p2"

	Convey("Test track of delete", t, func() {
		Convey("add", func() {
			JobMgr.AddTrackOfDelete(u1, p1, cydex.UPLOAD, true)
			JobMgr.AddTrackOfDelete(u2, p1, cydex.DOWNLOAD, true)
			JobMgr.AddTrackOfDelete(u2, p2, cydex.DOWNLOAD, true)

			So(JobMgr.track_deletes, ShouldHaveLength, 2)
		})

		Convey("get", func() {
			pids := JobMgr.GetTrackOfDelete(u2, cydex.DOWNLOAD, false, true)
			So(pids, ShouldHaveLength, 2)
			So(p1, ShouldBeIn, pids)
			So(p2, ShouldBeIn, pids)

			pids = JobMgr.GetTrackOfDelete(u1, cydex.UPLOAD, false, true)
			So(pids, ShouldHaveLength, 1)
			So(p1, ShouldBeIn, pids)

			pids = JobMgr.GetTrackOfDelete(u1, cydex.DOWNLOAD, false, true)
			So(pids, ShouldBeEmpty)
		})

		Convey("get and delete", func() {
			So(JobMgr.track_deletes, ShouldHaveLength, 2)

			pids := JobMgr.GetTrackOfDelete(u2, cydex.DOWNLOAD, true, true)
			So(pids, ShouldHaveLength, 2)
			So(p1, ShouldBeIn, pids)
			So(p2, ShouldBeIn, pids)

			So(JobMgr.track_deletes, ShouldHaveLength, 1)

			pids = JobMgr.GetTrackOfDelete(u1, cydex.UPLOAD, true, true)
			So(pids, ShouldHaveLength, 1)
			So(p1, ShouldBeIn, pids)

			So(JobMgr.track_deletes, ShouldBeEmpty)

			pids = JobMgr.GetTrackOfDelete(u1, cydex.DOWNLOAD, false, true)
			So(pids, ShouldBeEmpty)
		})
	})
}
