package task

import (
	trans "./.."
	// "./../../utils/cache"
	"./../models"
	"crypto/rand"
	"cydex"
	"cydex/transfer"
	"errors"
	"fmt"
	clog "github.com/cihub/seelog"
	// "github.com/garyburd/redigo/redis"
	// "github.com/pborman/uuid"
	"math"
	URL "net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	TASK_CACHE_TIMEOUT    = 20 * 60
	TASK_MAX_BITRATES_CNT = 1000
)

var (
	TaskMgr      *TaskManager
	taskno_start = uint32(0)
)

func init() {
	TaskMgr = NewTaskManager()
}

func GenerateTaskId() string {
	v := atomic.AddUint32(&taskno_start, 1)
	return fmt.Sprintf("%s-%d", TaskMgr.taskid_gen_prefix, v)
	// return uuid.New()
}

type TaskObserver interface {
	AddTask(t *Task)
	DelTask(t *Task)
	TaskStateNotify(t *Task, state *transfer.TaskState)
}

type UploadReq struct {
	*transfer.UploadTaskReq
	JobId         string
	LeftPkgSize   uint64 //issue-29 pkg未传输size
	FileSize      uint64
	restrict_mode int
}

// func NewUploadReq(r *transfer.UploadTaskReq, pid string) (req *UploadReq, err error) {
// 	if r == nil || pid == "" {
// 		return nil, errors.New("Invalid params")
// 	}
// 	req = &UploadReq{
// 		UploadTaskReq: r,
// 		Pid:           pid,
// 	}
// 	return req, nil
// }

type DownloadReq struct {
	*transfer.DownloadTaskReq
	// jzh: DownloadTaskReq里是owner_uid, TaskUid指请求的用户uid
	// TaskUid         string
	// meta  interface{}
	url   *URL.URL
	JobId string
}

// func NewDownloadReq(r *transfer.DownloadTaskReq, pid string, finished_sid_list []string) (req *DownloadReq, err error) {
// 	if r == nil || pid == "" {
// 		return nil, errors.New("Invalid params")
// 	}
// 	req = &DownloadReq{
// 		DownloadTaskReq: r,
// 		Pid:             pid,
// 		FinishedSidList: finished_sid_list,
// 	}
// 	return req, nil
// }

type Task struct {
	*models.Task
	bitrates     [TASK_MAX_BITRATES_CNT]uint64
	bitrates_cnt uint32
	timestamp    time.Time
}

// type Task struct {
// 	TaskId string `redis:"task_id"` // 任务ID
// 	JobId  string `redis:"job_id"`  // 包裹JobId
// 	Pid    string `redis:"pid"`
// 	Fid    string `redis:"fid"`
// 	Type   int    `redis:"type"` // 类型, U or D
// 	NumSeg int    `redis:"num_seg"`
// 	Nid    string `redis:"nid"`
// }

func NewTask(job_id string, msg *transfer.Message) *Task {
	t := new(Task)
	t.Task = new(models.Task)
	t.JobId = job_id
	t.timestamp = time.Now()

	var sid_list []string

	if msg.Cmd == "uploadtask" {
		req := msg.Req.UploadTask
		t.TaskId = req.TaskId
		// t.Uid = req.Uid
		t.Pid = req.Pid
		t.Fid = req.Fid
		t.Type = cydex.UPLOAD
		sid_list = req.SidList
	} else {
		req := msg.Req.DownloadTask
		t.TaskId = req.TaskId
		// t.Uid = ""
		t.Pid = req.Pid
		t.Fid = req.Fid
		t.Type = cydex.DOWNLOAD
		sid_list = req.SidList
	}

	t.NumSegs = len(sid_list)
	return t
}

func LoadTask(taskid string) *Task {
	if models.DB() == nil {
		return nil
	}
	task_m, _ := models.GetTask(taskid)
	if task_m == nil {
		return nil
	}
	t := &Task{Task: task_m}
	return t
}

func (self *Task) IsDispatched() bool {
	return self.NodeId != ""
}

func (self *Task) Stop() {
	clog.Infof("%s stop", self)
	if !self.IsDispatched() {
		return
	}

	const timeout = 5 * time.Second

	msg := transfer.NewReqMessage("", "stoptask", "", 0)
	msg.Req.StopTask = &transfer.StopTaskReq{
		TaskId: self.TaskId,
	}
	node := trans.NodeMgr.GetByNid(self.NodeId)
	if node != nil {
		if _, err := node.SendRequestSync(msg, timeout); err != nil {
			return
		}
	}
}

func (self *Task) inputBitrate(bitrate uint64) {
	idx := self.bitrates_cnt % TASK_MAX_BITRATES_CNT
	self.bitrates[idx] = bitrate
	self.bitrates_cnt++
}

// func (self *Task) IsOver() bool {
// 	finished_segs := 0
// 	for _, state := range self.SegsState {
// 		if state.State == "interrupted" || state.State == "end" {
// 			finished_segs++
// 		}
// 	}
// 	if finished_segs == len(self.SegsState) {
// 		return true
// 	}
// 	return false
// }

func (self *Task) String() string {
	var s string
	switch self.Type {
	case cydex.UPLOAD:
		s = "U"
	case cydex.DOWNLOAD:
		s = "D"
	default:
		s = "?"
	}
	id := self.TaskId
	if len(id) > 30 {
		id = id[:8]
	}
	return fmt.Sprintf("<Task(id:%s %s %d)>", id, s, self.NumSegs)
}

// 任务管理器
type TaskManager struct {
	scheduler         TaskScheduler
	lock              sync.Mutex
	observer_idx      uint32
	observers         map[uint32]TaskObserver
	cache_timeout     int
	taskid_gen_prefix string //issue-35
	tasks             map[string]*Task
}

func NewTaskManager() *TaskManager {
	t := new(TaskManager)
	t.tasks = make(map[string]*Task)
	t.observers = make(map[uint32]TaskObserver)
	// t.task_state_chan = make(chan *transfer.TaskState, 10)
	t.cache_timeout = TASK_CACHE_TIMEOUT

	b := make([]byte, 5)
	n, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	t.taskid_gen_prefix = fmt.Sprintf("%02x", b[:n])
	return t
}

func (self *TaskManager) SetScheduler(s TaskScheduler) {
	clog.Infof("Task Manager Set scheduler: %s", s.String())
	self.scheduler = s
}

func (self *TaskManager) AddObserver(o TaskObserver) (uint32, error) {
	if o == nil {
		return 0, errors.New("Observer is invalid")
	}
	defer func() {
		self.observer_idx++
		self.lock.Unlock()
	}()
	self.lock.Lock()
	self.observers[self.observer_idx] = o
	return self.observer_idx, nil
}

func (self *TaskManager) DelObserver(id uint32) {
	defer self.lock.Unlock()
	self.lock.Lock()
	delete(self.observers, id)
}

func (self *TaskManager) AddTask(t *Task) {
	// if err := SaveTaskToCache(t, self.cache_timeout); err != nil {
	// 	clog.Error("save task %s cache failed", t)
	// }

	var zoneid string
	node := trans.NodeMgr.GetByNid(t.NodeId)
	if node != nil {
		zoneid = node.ZoneId
		node.AddTaskCnt(t.Type, 1)
	}

	t.timestamp = time.Now()
	t.ZoneId = zoneid
	// issue-44: save task scheduler record
	if models.DB() != nil {
		models.CreateTask(t.Task)
	}

	self.lock.Lock()
	defer self.lock.Unlock()

	// 删除过期的task
	for k, v := range self.tasks {
		if time.Now().After(v.timestamp.Add(time.Duration(self.cache_timeout) * time.Second)) {
			clog.Infof("Delete timeout task %s", v)
			self.delTask(k)
		}
	}

	self.tasks[t.TaskId] = t
	for _, o := range self.observers {
		o.AddTask(t)
	}
	clog.Infof("Add task %s to node(%s)", t, t.NodeId)
}

func (self *TaskManager) GetTask(taskid string) *Task {
	// t, err := LoadTaskFromCache(taskid)
	// if err != nil {
	// 	clog.Error("load task(%s) from cache failed", taskid)
	// }
	// if t != nil {
	// 	UpdateTaskCache(t.TaskId, self.cache_timeout)
	// }
	// return t
	self.lock.Lock()
	defer self.lock.Unlock()

	t, ok := self.tasks[taskid]
	if !ok {
		t = LoadTask(taskid)
		self.tasks[taskid] = t
	}
	return t
}

func (self *TaskManager) delTask(taskid string) {
	t := self.tasks[taskid]
	delete(self.tasks, taskid)
	clog.Infof("Del task(%s) %+v, left:%d tasks", taskid, t, len(self.tasks))

	if t == nil {
		return
	}

	cnt := t.bitrates_cnt
	if cnt > TASK_MAX_BITRATES_CNT {
		cnt = TASK_MAX_BITRATES_CNT
	}
	min, avg, max, sd := BitrateStatistics(t.bitrates[:cnt])
	if models.DB() != nil {
		t.UpdateBitrateStatistics(min, avg, max, sd)
	}

	node := trans.NodeMgr.GetByNid(t.NodeId)
	if node != nil {
		node.AddTaskCnt(t.Type, -1)
	}

	for _, o := range self.observers {
		o.DelTask(t)
	}
}

func (self *TaskManager) DelTask(taskid string) {
	// t, err := LoadTaskFromCache(taskid)
	// if err != nil {
	// 	// do nothing
	// }
	//
	// DelTaskCache(taskid)
	self.lock.Lock()
	defer self.lock.Unlock()
	self.delTask(taskid)
}

func (self *TaskManager) DispatchUpload(req *UploadReq, timeout time.Duration) (rsp *transfer.Message, node *trans.Node, err error) {
	if req == nil || req.TaskId == "" {
		err = errors.New("Invalid req")
		return
	}
	if self.scheduler == nil {
		err = errors.New("No Scheduler, please set first")
		return
	}

	if node, err = self.scheduler.DispatchUpload(req); err != nil {
		return
	}
	if node == nil {
		err = errors.New("Scheduler didn't find suitable node for uploading")
		return
	}

	clog.Infof("try to dispatch task(U) %s to %s", req.TaskId, node)

	msg := transfer.NewReqMessage("", "uploadtask", "", 0)
	msg.Req.UploadTask = req.UploadTaskReq
	if rsp, err = node.SendRequestSync(msg, timeout); err != nil {
		return
	}

	if rsp.Rsp.Code != cydex.OK {
		err = fmt.Errorf("Response of uploadtask is error: %d", rsp.Rsp.Code)
		return
	}

	t := NewTask(req.JobId, msg)
	t.NodeId = node.Nid
	t.AllocatedBitrate = uint64(rsp.Rsp.UploadTask.RecomendBitrate)
	TaskMgr.AddTask(t)

	return
}

func (self *TaskManager) DispatchDownload(req *DownloadReq, timeout time.Duration) (rsp *transfer.Message, node *trans.Node, err error) {
	if req == nil || req.TaskId == "" {
		err = errors.New("Invalid req")
		return
	}
	if self.scheduler == nil {
		err = errors.New("No Scheduler, please set first")
		return
	}

	if node, err = self.scheduler.DispatchDownload(req); err != nil {
		return
	}
	if node == nil {
		err = errors.New("Scheduler didn't find suitable node for downloading")
		return
	}

	clog.Infof("try to dispatch task(D) %s to %s", req.TaskId, node)

	msg := transfer.NewReqMessage("", "downloadtask", "", 0)
	msg.Req.DownloadTask = req.DownloadTaskReq
	if rsp, err = node.SendRequestSync(msg, timeout); err != nil {
		return
	}
	if rsp.Rsp.Code != cydex.OK {
		err = fmt.Errorf("Response of downloadtask is error: %d", rsp.Rsp.Code)
		return
	}

	t := NewTask(req.JobId, msg)
	t.NodeId = node.Nid
	t.AllocatedBitrate = uint64(rsp.Rsp.DownloadTask.RecomendBitrate)
	TaskMgr.AddTask(t)
	return
}

func (self *TaskManager) handleTaskState(state *transfer.TaskState) (err error) {
	// clog.Tracef("TaskState: %+v", state)
	t := self.GetTask(state.TaskId)
	if t != nil {
		t.timestamp = time.Now()
	}

	if state.Sid != "" {
		if t != nil {
			t.inputBitrate(state.Bitrate)
		}
		self.lock.Lock()
		for _, o := range self.observers {
			o.TaskStateNotify(t, state)
		}
		self.lock.Unlock()
	} else {
		clog.Infof("%s is over", t)
		self.DelTask(state.TaskId)
	}

	var sv int
	s := strings.ToLower(state.State)
	switch s {
	case "transferring":
		sv = cydex.TRANSFER_STATE_DOING
	case "interrupted":
		sv = cydex.TRANSFER_STATE_PAUSE
	case "end":
		sv = cydex.TRANSFER_STATE_DONE
	default:
		return
	}
	if t != nil && t.State != sv {
		t.State = sv
		if models.DB() != nil {
			t.UpdateState(sv)
		}
	}

	return
}

// 侦听node接收的任务状态消息
func (self *TaskManager) TaskStateRoutine() {
	for states := range trans.NodeMgr.StateChan {
		for _, state := range states {
			self.handleTaskState(state)
		}
	}
}

// 根据相关信息停止任务
func (self *TaskManager) StopTasks(jobid string) {
	// tasks, err := LoadTasksByJobIdFromCache(jobid)
	tasks, err := models.GetTasksByJobId(jobid)
	if err != nil {
		clog.Errorf("load tasks by jobid(%s) error, %v", jobid, err)
		return
	}
	for _, tm := range tasks {
		t := &Task{Task: tm}
		go t.Stop()
	}
}

func (self *TaskManager) Scheduler() TaskScheduler {
	return self.scheduler
}

func (self *TaskManager) SetCacheTimeout(second int) {
	self.cache_timeout = second
}

func (self *TaskManager) LoadTasksFromCache(filter func(t *Task) bool) ([]*Task, error) {
	tasks := make([]*Task, 0, 10)

	self.lock.Lock()
	defer self.lock.Unlock()

	for _, t := range self.tasks {
		if filter != nil {
			if !filter(t) {
				continue
			}
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// func SaveTaskToCache(t *Task, timeout int) (err error) {
// 	conn := cache.Get()
// 	defer conn.Close()
// 	key := fmt.Sprintf("task#%s", t.TaskId)
// 	_, err = conn.Do("HMSET", redis.Args().Add(key).AddFlat(t)...)
// 	// _, err = conn.Do("HMSET", key, "task_id", t.TaskId, "job_id", t.JobId, "pid", t.Pid, "fid", t.Fid, "type", t.Type, "num_seg", t.NumSeg, "nid", t.Nid)
// 	if err != nil {
// 		return err
// 	}
// 	if _, err = conn.Do("EXPIRE", key, timeout); err != nil {
// 		return err
// 	}
// 	return nil
// }
//
// func UpdateTaskCache(task_id string, timeout int) error {
// 	conn := cache.Get()
// 	defer conn.Close()
// 	key := fmt.Sprintf("task#%s", task_id)
// 	if _, err := conn.Do("EXPIRE", key, timeout); err != nil {
// 		return err
// 	}
// 	return nil
// }
//
// func DelTaskCache(task_id string) error {
// 	conn := cache.Get()
// 	defer conn.Close()
// 	key := fmt.Sprintf("task#%s", task_id)
// 	if _, err := conn.Do("DEL", key); err != nil {
// 		return err
// 	}
// 	return nil
// }
//
// func LoadTaskFromCache(task_id string) (*Task, error) {
// 	conn := cache.Get()
// 	defer conn.Close()
// 	key := fmt.Sprintf("task#%s", task_id)
// 	t := new(Task)
// 	rsp, err := redis.Values(conn.Do("HGETALL", key))
// 	if err != nil {
// 		return nil, err
// 	}
// 	if err = redis.ScanStruct(rsp, t); err != nil {
// 		return nil, err
// 	}
// 	return t, nil
// }
//
// func LoadTasksByJobIdFromCache(job_id string) ([]*Task, error) {
// 	return LoadTasksFromCache(func(t *Task) bool {
// 		return t.JobId == job_id
// 	})
// }
//
// func LoadTasksByPidFromCache(pid string) ([]*Task, error) {
// 	return LoadTasksFromCache(func(t *Task) bool {
// 		return t.Pid == pid
// 	})
// }
//
// func LoadTasksFromCache(filter func(t *Task) bool) ([]*Task, error) {
// 	conn := cache.Get()
// 	defer conn.Close()
// 	keys, err := redis.Strings(conn.Do("KEYS", "task#*"))
// 	if err != nil {
// 		return nil, err
// 	}
// 	tasks := make([]*Task, 0, 10)
// 	for _, key := range keys {
// 		t, err := LoadTaskFromCache(key[5:])
// 		if err != nil {
// 			continue
// 		}
// 		if filter != nil {
// 			if !filter(t) {
// 				continue
// 			}
// 		}
// 		tasks = append(tasks, t)
// 	}
// 	return tasks, nil
// }

func LoadTasksFromCache(filter func(t *Task) bool) ([]*Task, error) {
	return TaskMgr.LoadTasksFromCache(filter)
}

func BitrateStatistics(bitrates []uint64) (min, avg, max uint64, sd float64) {
	if len(bitrates) == 0 {
		return
	}
	sum := uint64(0)
	for _, b := range bitrates {
		sum += b
		if min == 0 || b < min {
			min = b
		}
		if b > max {
			max = b
		}
	}
	avg = sum / uint64(len(bitrates))

	// 计算标准差
	sum2 := float64(0)
	for _, b := range bitrates {
		v := float64(b) - float64(avg)
		sum2 += math.Pow(v, 2)
	}
	sd = math.Sqrt(sum2)
	return
}
