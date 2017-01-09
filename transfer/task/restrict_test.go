package task

import (
	trans "./../"
	"./../models"
	"cydex"
	"cydex/transfer"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func Test_XidResource(t *testing.T) {
	Convey("Test XidResource", t, func() {
		X := NewXidResource()
		X.Add("x1", "n1", 1*time.Second)
		So(X.Len(), ShouldEqual, 1)
		time.Sleep(5)
		X.DelExpired()
		So(X.Len(), ShouldEqual, 1)
		time.Sleep(1100 * time.Millisecond)
		X.DelExpired()
		So(X.Len(), ShouldEqual, 0)
	})
}

func Test_RestrictUploadScheduler(t *testing.T) {
	// var err error
	Convey("Test Restrict", t, func() {
		Convey("Test restrict by pid", func() {
			S := NewRestrictUploadScheduler(TASK_RESTRICT_BY_PID)
			node := &trans.Node{
				Nid: "n1",
				Info: trans.NodeInfo{
					FreeStorage: 10000,
				},
			}
			trans.NodeMgr.AddNode(node)

			t1 := &Task{
				Task: &models.Task{
					TaskId: "t0",
					JobId:  "jobid",
					Pid:    "1234567890ab1111122222",
					Type:   cydex.UPLOAD,
					NodeId: "n1",
					Fid:    "1234567890ab111112222201",
				},
			}
			S.AddTask(t1)

			r1 := &UploadReq{
				UploadTaskReq: &transfer.UploadTaskReq{
					TaskId: "t1",
					Uid:    "1234567890ab",
					Pid:    "1234567890ab1111122222",
					Fid:    "1234567890ab111112222202",
				},
			}
			n, err := S.DispatchUpload(r1)
			So(err, ShouldBeNil)
			So(n, ShouldEqual, node)

			r2 := &UploadReq{
				UploadTaskReq: &transfer.UploadTaskReq{
					TaskId: "t2",
					Uid:    "1234567890ab",
					Pid:    "1234567890ab1111122223",
					Fid:    "1234567890ab111112222301",
				},
			}
			n, err = S.DispatchUpload(r2)
			So(err, ShouldBeNil)
			So(n, ShouldBeNil)
		})

		Convey("Test restrict by fid", func() {
			S := NewRestrictUploadScheduler(TASK_RESTRICT_BY_FID)
			node := &trans.Node{
				Nid: "n1",
				Info: trans.NodeInfo{
					FreeStorage: 10000,
				},
			}
			trans.NodeMgr.AddNode(node)
			t1 := &Task{
				Task: &models.Task{
					TaskId: "t0",
					Type:   cydex.UPLOAD,
					Pid:    "pid",
					Fid:    "1234567890ab111112222201",
					NodeId: "n1",
				},
			}
			S.AddTask(t1)

			r1 := &UploadReq{
				UploadTaskReq: &transfer.UploadTaskReq{
					TaskId: "t1",
					Uid:    "1234567890ab",
					Fid:    "1234567890ab111112222202",
				},
			}
			n, err := S.DispatchUpload(r1)
			So(err, ShouldBeNil)
			So(n, ShouldBeNil)
		})

		Convey("Test restrict by fid with size", func() {
			S := NewRestrictUploadScheduler(TASK_RESTRICT_BY_FID)
			node := &trans.Node{
				Nid: "n1",
				Info: trans.NodeInfo{
					FreeStorage: 128,
				},
			}
			trans.NodeMgr.AddNode(node)
			t1 := &Task{
				Task: &models.Task{
					TaskId: "t0",
					JobId:  "jobid",
					Fid:    "1234567890ab111112222201",
					Type:   cydex.UPLOAD,
					NodeId: "n1",
				},
			}
			S.AddTask(t1)

			r1 := &UploadReq{
				LeftPkgSize: 1234,
				FileSize:    127,
				UploadTaskReq: &transfer.UploadTaskReq{
					TaskId: "t1",
					Uid:    "1234567890ab",
					Fid:    "1234567890ab111112222201",
					Size:   127,
				},
			}
			n, err := S.DispatchUpload(r1)
			So(err, ShouldBeNil)
			So(n, ShouldEqual, node)
			So(r1.restrict_mode, ShouldEqual, TASK_RESTRICT_BY_FID)

			r2 := &UploadReq{
				LeftPkgSize: 1234,
				FileSize:    129,
				UploadTaskReq: &transfer.UploadTaskReq{
					TaskId: "t2",
					Uid:    "1234567890ab",
					Fid:    "1234567890ab111112222201",
					Size:   129,
				},
			}
			n, err = S.DispatchUpload(r2)
			So(err, ShouldBeNil)
			So(n, ShouldBeNil)
		})

		Convey("Test restrict by pid with size", func() {
			S := NewRestrictUploadScheduler(TASK_RESTRICT_BY_PID)
			node := &trans.Node{
				Nid: "n1",
				Info: trans.NodeInfo{
					FreeStorage: 1000,
				},
			}
			trans.NodeMgr.AddNode(node)
			t1 := &Task{
				Task: &models.Task{
					TaskId: "t0",
					JobId:  "jobid",
					Fid:    "1234567890ab111112222201",
					Type:   cydex.UPLOAD,
					NodeId: "n1",
				},
			}
			S.AddTask(t1)

			r1 := &UploadReq{
				LeftPkgSize: 999,
				FileSize:    127,
				UploadTaskReq: &transfer.UploadTaskReq{
					TaskId: "t1",
					Uid:    "1234567890ab",
					Fid:    "1234567890ab111112222201",
					Size:   127,
				},
			}
			So(r1.restrict_mode, ShouldEqual, 0)
			n, err := S.DispatchUpload(r1)
			So(err, ShouldBeNil)
			So(n, ShouldEqual, node)
			So(r1.restrict_mode, ShouldEqual, TASK_RESTRICT_BY_PID)

			r2 := &UploadReq{
				LeftPkgSize: 1001,
				FileSize:    129,
				UploadTaskReq: &transfer.UploadTaskReq{
					TaskId: "t2",
					Uid:    "1234567890ab",
					Fid:    "1234567890ab111112222201",
					Size:   129,
				},
			}
			n, err = S.DispatchUpload(r2)
			So(err, ShouldBeNil)
			So(n, ShouldBeNil)
		})
	})
}
