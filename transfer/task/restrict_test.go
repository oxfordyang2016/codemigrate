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
		X.Add("x1", trans.NewNode(nil, nil), 1*time.Second)
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
				Node: &models.Node{
					Nid: "n1",
				},
			}
			t1 := &Task{
				TaskId: "t0",
				Type:   cydex.UPLOAD,
				Node:   node,
				UploadReq: &UploadReq{
					UploadTaskReq: &transfer.UploadTaskReq{
						TaskId: "t1",
						Uid:    "1234567890ab",
						Pid:    "1234567890ab1111122222",
						Fid:    "1234567890ab111112222201",
					},
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
				Node: &models.Node{
					Nid: "n1",
				},
			}
			t1 := &Task{
				TaskId: "t0",
				Type:   cydex.UPLOAD,
				Node:   node,
				UploadReq: &UploadReq{
					UploadTaskReq: &transfer.UploadTaskReq{
						TaskId: "t0",
						Uid:    "1234567890ab",
						Fid:    "1234567890ab111112222201",
					},
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
				Node: &models.Node{
					Nid: "n1",
				},
				Info: trans.NodeInfo{
					FreeStorage: 128,
				},
			}
			t1 := &Task{
				TaskId: "t0",
				Type:   cydex.UPLOAD,
				Node:   node,
				UploadReq: &UploadReq{
					UploadTaskReq: &transfer.UploadTaskReq{
						TaskId: "t0",
						Uid:    "1234567890ab",
						Fid:    "1234567890ab111112222201",
						Size:   11,
					},
				},
			}
			S.AddTask(t1)

			r1 := &UploadReq{
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

			r2 := &UploadReq{
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
