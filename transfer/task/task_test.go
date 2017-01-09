package task

import (
	// trans "./../"
	// "./../../utils/cache"
	"./../models"
	"cydex"
	"cydex/transfer"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func init() {
	// cache.Init("redis://:MyCydex@127.0.0.1:6379", 3, 240)
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
	fo := new(FakeTaskObserver)
	Convey("Test TaskMgr", t, func() {
		Convey("Add task", func() {
			TaskMgr.AddObserver(fo)
			t := &Task{
				Task: &models.Task{
					TaskId: "t0",
					Type:   cydex.UPLOAD,
				},
			}
			TaskMgr.AddTask(t)
			So(fo.cnt, ShouldEqual, 1)

			tasks, err := LoadTasksFromCache(nil)
			So(err, ShouldBeNil)
			So(tasks, ShouldHaveLength, 1)

			So(t.NodeId, ShouldBeEmpty)
			So(t.IsDispatched(), ShouldBeFalse)
		})
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
		Convey("Del task", func() {
			t := TaskMgr.tasks["t0"]
			So(t, ShouldNotBeNil)

			TaskMgr.DelTask("t0")
			So(fo.cnt, ShouldEqual, 0)

			t = TaskMgr.tasks["t0"]
			So(t, ShouldBeNil)
		})
	})
}

func Test_BitrateStatistics(t *testing.T) {
	Convey("bitrate statistics", t, func() {
		Convey("all are 0", func() {
			brs := []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0}
			min, avg, max, sd := BitrateStatistics(brs)
			So(min, ShouldEqual, 0)
			So(avg, ShouldEqual, 0)
			So(max, ShouldEqual, 0)
			So(sd, ShouldAlmostEqual, 0.0, .0001)
		})
		Convey("empty or nil", func() {
			min, avg, max, sd := BitrateStatistics([]uint64{})
			So(min, ShouldEqual, 0)
			So(avg, ShouldEqual, 0)
			So(max, ShouldEqual, 0)
			So(sd, ShouldAlmostEqual, 0.0, .0001)

			min, avg, max, sd = BitrateStatistics(nil)
			So(min, ShouldEqual, 0)
			So(avg, ShouldEqual, 0)
			So(max, ShouldEqual, 0)
			So(sd, ShouldAlmostEqual, 0.0, .0001)
		})
		Convey("some data", func() {
		})
	})
}
