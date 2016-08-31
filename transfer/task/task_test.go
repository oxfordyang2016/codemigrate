package task

import (
	// trans "./../"
	// "./../models"
	"./../../utils/cache"
	"cydex"
	"cydex/transfer"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func init() {
	cache.Init("redis://:MyCydex@127.0.0.1:6379", 3, 240)
}

type FakeTaskObserver struct {
	cnt        int
	update_cnt int
}

func (self *FakeTaskObserver) AddTask(t *Task) {
	self.cnt++
}

func (self *FakeTaskObserver) DelTask(t *Task) {
	self.cnt--
}

func (self *FakeTaskObserver) TaskStateNotify(t *Task, state *transfer.TaskState) {
	self.update_cnt++
}

func Test_TaskManager(t *testing.T) {

	Convey("Test TaskMgr", t, func() {
		Convey("task controller and observer", func() {
			fo := new(FakeTaskObserver)
			TaskMgr.AddObserver(fo)
			t1 := &Task{
				TaskId: "t0",
				Type:   cydex.UPLOAD,
			}
			TaskMgr.AddTask(t1)
			So(fo.cnt, ShouldEqual, 1)
			TaskMgr.DelTask("t0")
			So(fo.cnt, ShouldEqual, 0)

			So(t1.Nid, ShouldBeEmpty)
			So(t1.IsDispatched(), ShouldBeFalse)

			Convey("handle task state", func() {
				state := &transfer.TaskState{
					TaskId:     "t0",
					State:      "transferring",
					Sid:        "s1",
					TotalBytes: 123,
					Bitrate:    1000,
				}
				So(fo.update_cnt, ShouldEqual, 0)
				TaskMgr.handleTaskState(state)
				So(fo.update_cnt, ShouldEqual, 1)
			})
		})
	})
}
