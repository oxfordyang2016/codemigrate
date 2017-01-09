package statistics

import (
	"./../transfer/task"
	"./models"
	"cydex"
	"cydex/transfer"
	// clog "github.com/cihub/seelog"
	"sync"
)

type BitrateArray []uint64

// 任务统计
type StatTask struct {
	Type             int
	segs_total_bytes map[string]uint64
	bitrates         []uint64
}

func NewStatTask() *StatTask {
	o := new(StatTask)
	o.segs_total_bytes = make(map[string]uint64)
	return o
}

func (self *StatTask) TotalBytes() uint64 {
	var total uint64
	for _, b := range self.segs_total_bytes {
		total += b
	}
	return total
}

type StatTransferManager struct {
	task_mux   sync.Mutex
	stat_tasks map[string]*StatTask
}

func calcMinMaxAvg(array []uint64) (max, min, avg uint64) {
	if len(array) == 0 {
		return 0, 0, 0
	}

	max = array[0]
	min = array[0]
	total := uint64(0)
	for _, a := range array {
		total += a
		if a > max {
			max = a
		}
		if min == 0 || (a > 0 && a < min) {
			min = a
		}
	}
	avg = total / uint64(len(array))
	return
}

func NewStatTransferManager() *StatTransferManager {
	o := new(StatTransferManager)
	o.stat_tasks = make(map[string]*StatTask)
	return o
}

func (self *StatTransferManager) GetStat(typ int) *cydex.TransferStat {
	trans, _ := models.GetTransfers()
	if trans == nil {
		return nil
	}

	var (
		totalbytes, min, max, total_avg, total_tasks               uint64
		pTotalBytes, pMaxBitrate, pMinBitrate, pAvgBitrate, pTasks *uint64
	)

	cnt := uint64(0)
	for i, model := range trans {
		switch typ {
		case cydex.UPLOAD:
			pTotalBytes = &model.RxTotalBytes
			pMaxBitrate = &model.RxMaxBitrate
			pMinBitrate = &model.RxMinBitrate
			pAvgBitrate = &model.RxAvgBitrate
			pTasks = &model.RxTasks
		case cydex.DOWNLOAD:
			pTotalBytes = &model.TxTotalBytes
			pMaxBitrate = &model.TxMaxBitrate
			pMinBitrate = &model.TxMinBitrate
			pAvgBitrate = &model.TxAvgBitrate
			pTasks = &model.TxTasks
		default:
			break
		}

		cnt++

		if i == 0 {
			max = *pMaxBitrate
			min = *pMinBitrate
		} else {
			if *pMaxBitrate > max {
				max = *pMaxBitrate
			}
			if min == 0 {
				min = *pMinBitrate
			} else {
				if *pMinBitrate > 0 && *pMinBitrate < min {
					min = *pMinBitrate
				}
			}
		}
		totalbytes += *pTotalBytes
		total_avg += *pAvgBitrate
		total_tasks += *pTasks
	}

	ret := &cydex.TransferStat{
		TotalBytes: totalbytes,
		MaxBitrate: max,
		MinBitrate: min,
		AvgBitrate: total_avg / cnt,
		TotalTasks: total_tasks,
	}
	return ret
}

func (self *StatTransferManager) GetNodeStat(node_id string, typ int) *cydex.TransferStat {
	stat_trans, _ := models.GetTransfer(node_id)
	if stat_trans == nil {
		return nil
	}

	ret := new(cydex.TransferStat)
	switch typ {
	case cydex.UPLOAD:
		ret.TotalBytes = stat_trans.RxTotalBytes
		ret.MaxBitrate = stat_trans.RxMaxBitrate
		ret.MinBitrate = stat_trans.RxMinBitrate
		ret.AvgBitrate = stat_trans.RxAvgBitrate
		ret.TotalTasks = stat_trans.RxTasks
	case cydex.DOWNLOAD:
		ret.TotalBytes = stat_trans.TxTotalBytes
		ret.MaxBitrate = stat_trans.TxMaxBitrate
		ret.MinBitrate = stat_trans.TxMinBitrate
		ret.AvgBitrate = stat_trans.TxAvgBitrate
		ret.TotalTasks = stat_trans.TxTasks
	default:
		return nil
	}
	return ret
}

// implement trans.TaskObserver
func (self *StatTransferManager) AddTask(t *task.Task) {
	if t == nil {
		return
	}
	stat_trans, _ := models.GetTransfer(t.NodeId)
	if stat_trans == nil {
		stat_trans, _ = models.NewTransferWithId(t.NodeId)
	}
	if stat_trans == nil {
		// log
		return
	}
	switch t.Type {
	case cydex.UPLOAD:
		stat_trans.RxTasks++
	case cydex.DOWNLOAD:
		stat_trans.TxTasks++
	default:
		return
	}
	if err := stat_trans.Update(); err != nil {
		// log
	}
}

func (self *StatTransferManager) DelTask(t *task.Task) {
	if t == nil {
		return
	}

	self.task_mux.Lock()
	defer self.task_mux.Unlock()

	stat_task := self.stat_tasks[t.TaskId]
	if stat_task == nil || len(stat_task.bitrates) == 0 {
		return
	}
	stat_trans, _ := models.GetTransfer(t.NodeId)
	if stat_trans == nil {
		return
	}

	max, min, avg := calcMinMaxAvg(stat_task.bitrates)

	var pTotalBytes, pMaxBitrate, pMinBitrate, pAvgBitrate, pTasks *uint64
	switch t.Type {
	case cydex.UPLOAD:
		pTotalBytes = &stat_trans.RxTotalBytes
		pMaxBitrate = &stat_trans.RxMaxBitrate
		pMinBitrate = &stat_trans.RxMinBitrate
		pAvgBitrate = &stat_trans.RxAvgBitrate
		pTasks = &stat_trans.RxTasks
	case cydex.DOWNLOAD:
		pTotalBytes = &stat_trans.TxTotalBytes
		pMaxBitrate = &stat_trans.TxMaxBitrate
		pMinBitrate = &stat_trans.TxMinBitrate
		pAvgBitrate = &stat_trans.TxAvgBitrate
		pTasks = &stat_trans.TxTasks
	default:
		return
	}

	*pTotalBytes += stat_task.TotalBytes()
	if max > *pMaxBitrate {
		*pMaxBitrate = max
	}
	if *pMinBitrate == 0 || (min > 0 && min < *pMinBitrate) {
		*pMinBitrate = min
	}

	if *pTasks <= 1 {
		*pAvgBitrate = avg
	} else {
		*pAvgBitrate = (*pAvgBitrate*(*pTasks-1) + avg) / (*pTasks)
	}
	stat_trans.Update()

	delete(self.stat_tasks, t.TaskId)
}

func (self *StatTransferManager) TaskStateNotify(t *task.Task, state *transfer.TaskState) {
	if state == nil {
		return
	}
	if state.Sid == "" || state.State != "end" {
		return
	}

	self.task_mux.Lock()
	defer self.task_mux.Unlock()

	stat_task := self.stat_tasks[state.TaskId]
	if stat_task == nil {
		stat_task = NewStatTask()
		if t == nil {
			return
		}
		stat_task.Type = t.Type
		self.stat_tasks[state.TaskId] = stat_task
	}
	stat_task.segs_total_bytes[state.Sid] = state.GetTotalBytes()
	stat_task.bitrates = append(stat_task.bitrates, state.Bitrate)
}

func (self *StatTransferManager) GetCurBitrate(typ int) uint64 {
	self.task_mux.Lock()
	defer self.task_mux.Unlock()
	var ret uint64
	for _, st := range self.stat_tasks {
		if st.Type != typ {
			continue
		}
		if len(st.bitrates) > 0 {
			ret += st.bitrates[len(st.bitrates)-1]
		}
	}
	return ret
}
