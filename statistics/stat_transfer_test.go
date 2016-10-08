package statistics

import (
	"./../transfer/task"
	"./../utils/db"
	"./models"
	"cydex"
	"cydex/transfer"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
)

const (
	// TEST_DB = ":memory:"
	TEST_DB = "/tmp/statistics_test.db"
)

var (
	StatTransferMgr *StatTransferManager
)

func init() {
	if TEST_DB != ":memory:" {
		os.Remove(TEST_DB)
	}
	db.CreateEngine("sqlite3", TEST_DB, false)
	models.SyncTables()

	StatTransferMgr = NewStatTransferManager()
}

func Test_GetStat(t *testing.T) {
	transfers := []*models.Transfer{
		&models.Transfer{NodeId: "n_1", RxTotalBytes: 5000, RxMaxBitrate: 1000, RxMinBitrate: 500, RxAvgBitrate: 100, RxTasks: 10, TxTotalBytes: 7000, TxMaxBitrate: 9001, TxMinBitrate: 0, TxAvgBitrate: 300, TxTasks: 20},
		&models.Transfer{NodeId: "n_2", RxTotalBytes: 2000, RxMaxBitrate: 2100, RxMinBitrate: 2, RxAvgBitrate: 200, RxTasks: 9, TxTotalBytes: 3000, TxMaxBitrate: 4000, TxMinBitrate: 50, TxAvgBitrate: 100, TxTasks: 15},
		&models.Transfer{NodeId: "n_3", RxTotalBytes: 3100, RxMaxBitrate: 3000, RxMinBitrate: 107, RxAvgBitrate: 300, RxTasks: 8, TxTotalBytes: 4050, TxMaxBitrate: 2009, TxMinBitrate: 766, TxAvgBitrate: 302, TxTasks: 13},
		&models.Transfer{NodeId: "n_4", RxTotalBytes: 4010, RxMaxBitrate: 4001, RxMinBitrate: 300, RxAvgBitrate: 400, RxTasks: 7, TxTotalBytes: 2009, TxMaxBitrate: 1235, TxMinBitrate: 900, TxAvgBitrate: 400, TxTasks: 27},
	}
	for _, t := range transfers {
		models.NewTransfer(t)
	}

	Convey("Test GetStat", t, func() {
		Convey("get single stat", func() {
			ret := StatTransferMgr.GetNodeStat("n_2", cydex.DOWNLOAD)
			So(ret, ShouldNotBeNil)
			So(ret.TotalBytes, ShouldEqual, 3000)
			So(ret.MaxBitrate, ShouldEqual, 4000)
			So(ret.MinBitrate, ShouldEqual, 50)
			So(ret.AvgBitrate, ShouldEqual, 100)
			So(ret.TotalTasks, ShouldEqual, 15)

			ret = StatTransferMgr.GetNodeStat("n_3", cydex.UPLOAD)
			So(ret, ShouldNotBeNil)
			So(ret.TotalBytes, ShouldEqual, 3100)
			So(ret.MaxBitrate, ShouldEqual, 3000)
			So(ret.MinBitrate, ShouldEqual, 107)
			So(ret.AvgBitrate, ShouldEqual, 300)
			So(ret.TotalTasks, ShouldEqual, 8)

			ret = StatTransferMgr.GetNodeStat("nodeid not existed", cydex.UPLOAD)
			So(ret, ShouldBeNil)
		})
		Convey("get outline stat", func() {
			ret := StatTransferMgr.GetStat(cydex.UPLOAD)
			So(ret, ShouldNotBeNil)
			So(ret.TotalBytes, ShouldEqual, 5000+2000+3100+4010)
			So(ret.MaxBitrate, ShouldEqual, 4001)
			So(ret.MinBitrate, ShouldEqual, 2)
			So(ret.AvgBitrate, ShouldEqual, 250)
			So(ret.TotalTasks, ShouldEqual, 10+9+8+7)

			ret = StatTransferMgr.GetStat(cydex.DOWNLOAD)
			So(ret, ShouldNotBeNil)
			So(ret.TotalBytes, ShouldEqual, 7000+3000+4050+2009)
			So(ret.MaxBitrate, ShouldEqual, 9001)
			So(ret.MinBitrate, ShouldEqual, 50)
			So(ret.AvgBitrate, ShouldEqual, (300+100+302+400)/4)
			So(ret.TotalTasks, ShouldEqual, 20+15+13+27)

		})
	})
}

func Test_HandleTask(t *testing.T) {
	Convey("Test HandleTask", t, func() {
		Convey("Add Task", func() {
			t := &task.Task{
				TaskId: "t1",
				Type:   cydex.UPLOAD,
				Nid:    "n1",
			}
			StatTransferMgr.AddTask(t)
			tr, _ := models.GetTransfer("n1")
			So(tr, ShouldNotBeNil)
			So(tr.RxTasks, ShouldEqual, 1)
			So(tr.TxTasks, ShouldEqual, 0)

			t2 := &task.Task{
				TaskId: "t2",
				Type:   cydex.UPLOAD,
				Nid:    "n1",
			}
			StatTransferMgr.AddTask(t2)
			tr, _ = models.GetTransfer("n1")
			So(tr, ShouldNotBeNil)
			So(tr.RxTasks, ShouldEqual, 2)
			So(tr.TxTasks, ShouldEqual, 0)
		})

		Convey("handle Task state", func() {
			task := &task.Task{
				TaskId: "t1",
				Type:   cydex.UPLOAD,
				Nid:    "n1",
			}
			states := []*transfer.TaskState{
				&transfer.TaskState{TaskId: "t1", TotalBytes: 10, Bitrate: 1000},
				&transfer.TaskState{TaskId: "t1", TotalBytes: 20, Bitrate: 2000},
				&transfer.TaskState{TaskId: "t1", TotalBytes: 30, Bitrate: 3000},
				&transfer.TaskState{TaskId: "t1", TotalBytes: 40, Bitrate: 4000},
			}

			for _, s := range states {
				StatTransferMgr.TaskStateNotify(task, s)
			}
			So(StatTransferMgr.stat_tasks, ShouldHaveLength, 1)
			stat_t := StatTransferMgr.stat_tasks["t1"]
			So(stat_t, ShouldNotBeNil)
			So(stat_t.totalbytes, ShouldEqual, 40)
			So(stat_t.bitrates, ShouldHaveLength, 4)
		})

		Convey("handle Del task", func() {
			t := &task.Task{
				TaskId: "t1",
				Type:   cydex.UPLOAD,
				Nid:    "n1",
			}
			StatTransferMgr.DelTask(t)
			tr, _ := models.GetTransfer("n1")
			So(tr, ShouldNotBeNil)
			So(tr.RxTasks, ShouldEqual, 2)
			So(tr.TxTasks, ShouldEqual, 0)
			So(tr.RxMaxBitrate, ShouldEqual, 4000)
			So(tr.RxMinBitrate, ShouldEqual, 1000)
			So(tr.RxAvgBitrate, ShouldEqual, 2500)
			So(tr.RxTotalBytes, ShouldEqual, 40)
		})
	})
}
