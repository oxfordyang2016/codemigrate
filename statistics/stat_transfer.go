package statistics

import (
	// trans "./../transfer"
	"./../transfer/task"
	"./models"
	"cydex"
	"cydex/transfer"
	// "fmt"
)

type BitrateArray []uint64

// 任务统计
type StatTask struct {
	totalbytes uint64
	bitrates   []uint64
}

type StatTransferManager struct {
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
	stat_trans, _ := models.GetTransfer(t.Nid)
	if stat_trans == nil {
		stat_trans, _ = models.NewTransferWithId(t.Nid)
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

	stat_task := self.stat_tasks[t.TaskId]
	if stat_task == nil || len(stat_task.bitrates) == 0 {
		return
	}
	stat_trans, _ := models.GetTransfer(t.Nid)
	if stat_trans == nil {
		return
	}

	max, min, avg := calcMinMaxAvg(stat_task.bitrates)

	var pTotalBytes, pMaxBitrate, pMinBitrate, pAvgBitrate *uint64
	switch t.Type {
	case cydex.UPLOAD:
		pTotalBytes = &stat_trans.RxTotalBytes
		pMaxBitrate = &stat_trans.RxMaxBitrate
		pMinBitrate = &stat_trans.RxMinBitrate
		pAvgBitrate = &stat_trans.RxAvgBitrate
	case cydex.DOWNLOAD:
		pTotalBytes = &stat_trans.TxTotalBytes
		pMaxBitrate = &stat_trans.TxMaxBitrate
		pMinBitrate = &stat_trans.TxMinBitrate
		pAvgBitrate = &stat_trans.TxAvgBitrate
	default:
		return
	}

	// FIXME 对下载来说TotalBytes比实际文件要小
	*pTotalBytes += stat_task.totalbytes
	if max > *pMaxBitrate {
		*pMaxBitrate = max
	}
	if *pMinBitrate == 0 || (min > 0 && min < *pMinBitrate) {
		*pMinBitrate = min
	}

	if *pAvgBitrate == 0 {
		*pAvgBitrate = avg
	} else {
		// NOTE: 和原均值平分
		*pAvgBitrate = (*pAvgBitrate + avg) / 2
	}
	stat_trans.Update()

	delete(self.stat_tasks, t.TaskId)
}

func (self *StatTransferManager) TaskStateNotify(t *task.Task, state *transfer.TaskState) {
	if state == nil {
		return
	}
	stat_task := self.stat_tasks[state.TaskId]
	if stat_task == nil {
		stat_task = new(StatTask)
		self.stat_tasks[state.TaskId] = stat_task
	}
	stat_task.totalbytes = state.GetTotalBytes()
	stat_task.bitrates = append(stat_task.bitrates, state.Bitrate)
}
