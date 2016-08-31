package task

import (
	trans "./.."
	"./../../utils/cache"
	"cydex"
	"cydex/transfer"
	"errors"
	"fmt"
	clog "github.com/cihub/seelog"
	"github.com/garyburd/redigo/redis"
	"github.com/pborman/uuid"
	"sync"
	"time"
)

const (
	TASK_CACHE_TIMEOUT = 20 * 60
)

var (
	TaskMgr *TaskManager
)

func init() {
	TaskMgr = NewTaskManager()
}

func GenerateTaskId() string {
	return uuid.New()
}

type TaskObserver interface {
	AddTask(t *Task)
	DelTask(t *Task)
	TaskStateNotify(t *Task, state *transfer.TaskState)
}

type UploadReq struct {
	*transfer.UploadTaskReq
	JobId         string
	PkgSize       uint64
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
	FinishedSidList []string //已经下完的片段
	meta            interface{}
	JobId           string
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
	TaskId string `redis:"task_id"` // 任务ID
	JobId  string `redis:"job_id"`  // 包裹JobId
	// Uid         string
	Pid  string `redis:"pid"`
	Fid  string `redis:"fid"`
	Type int    `redis:"type"` // 类型, U or D
	// UploadReq   *UploadReq
	// DownloadReq *DownloadReq
	// SegsState   map[string]*transfer.TaskState
	// CreateAt time.Time
	// UpdateAt time.Time
	NumSeg int    `redis:"num_seg"`
	Nid    string `redis:"nid"`
}

func NewTask(job_id string, msg *transfer.Message) *Task {
	t := new(Task)
	t.JobId = job_id
	// t.CreateAt = time.Now()
	// t.UpdateAt = time.Now()
	// t.SegsState = make(map[string]*transfer.TaskState)

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

	t.NumSeg = len(sid_list)
	return t
}

func (self *Task) IsDispatched() bool {
	return self.Nid != ""
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
	node := trans.NodeMgr.GetByNid(self.Nid)
	if node != nil {
		if _, err := node.SendRequestSync(msg, timeout); err != nil {
			return
		}
	}
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
	if len(id) > 8 {
		id = id[:8]
	}
	return fmt.Sprintf("<Task(id:%s %s %d)>", id, s, self.NumSeg)
}

// 任务管理器
type TaskManager struct {
	scheduler TaskScheduler
	lock      sync.Mutex
	// tasks        map[string]*Task // task_id -> task
	observer_idx uint32
	observers    map[uint32]TaskObserver
	// task_state_routine_run bool
	// task_state_chan        chan *transfer.TaskState
	cache_timeout int
}

func NewTaskManager() *TaskManager {
	t := new(TaskManager)
	// t.tasks = make(map[string]*Task)
	t.observers = make(map[uint32]TaskObserver)
	// t.task_state_chan = make(chan *transfer.TaskState, 10)
	t.cache_timeout = TASK_CACHE_TIMEOUT
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
	// delete timeouted task
	// for _, t := range self.tasks {
	// 	if time.Since(t.UpdateAt) > TASK_TIMEOUT {
	// 		self.DelTask(t.TaskId)
	// 	}
	// }
	if err := SaveTaskToCache(t, self.cache_timeout); err != nil {
		clog.Error("save task %s cache failed", t)
	}

	self.lock.Lock()
	defer self.lock.Unlock()

	// self.tasks[t.TaskId] = t
	for _, o := range self.observers {
		o.AddTask(t)
	}
	clog.Infof("Add task %s", t)
}

func (self *TaskManager) GetTask(taskid string) *Task {
	t, err := LoadTaskFromCache(taskid)
	if err != nil {
		clog.Error("load task(%s) from cache failed", taskid)
	}
	if t != nil {
		UpdateTaskCache(t.TaskId, self.cache_timeout)
	}
	return t
	// defer self.lock.Unlock()
	// self.lock.Lock()
	// t, _ = self.tasks[taskid]
	// return
}

func (self *TaskManager) DelTask(taskid string) {
	t, err := LoadTaskFromCache(taskid)
	if err != nil {
		// do nothing
	}

	DelTaskCache(taskid)

	self.lock.Lock()
	defer self.lock.Unlock()
	if t != nil {
		for _, o := range self.observers {
			o.DelTask(t)
		}
	}
	clog.Infof("Del task(%s) %+v", taskid, t)
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
		err = errors.New("Scheduler didn't find suitable node!")
		return
	}

	msg := transfer.NewReqMessage("", "uploadtask", "", 0)
	msg.Req.UploadTask = req.UploadTaskReq
	if rsp, err = node.SendRequestSync(msg, timeout); err != nil {
		// log
		// clog.Errorf("%s send upload task failed, err:%s", node, err)
		return
	}

	if rsp.Rsp.Code != cydex.OK {
		// clog.Errorf("%s send upload task failed, code:%d", node, rsp.Rsp.Code)
		return
	}

	t := NewTask(req.JobId, msg)
	t.Nid = node.Nid
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
		// Log
		return
	}

	msg := transfer.NewReqMessage("", "downloadtask", "", 0)
	msg.Req.DownloadTask = req.DownloadTaskReq
	if rsp, err = node.SendRequestSync(msg, timeout); err != nil {
		// log
		return
	}
	if rsp.Rsp.Code != cydex.OK {
		return
	}

	t := NewTask(req.JobId, msg)
	t.Nid = node.Nid
	TaskMgr.AddTask(t)
	return
}

func (self *TaskManager) handleTaskState(state *transfer.TaskState) (err error) {
	// clog.Tracef("TaskState: %+v", state)
	t := self.GetTask(state.TaskId)
	// clog.Trace(t)
	// if t != nil {
	// 	t.UpdateAt = time.Now()
	// 	s := t.SegsState[state.Sid]
	// 	if s != nil {
	// 		s.State = state.State
	// 		s.TotalBytes = state.TotalBytes
	// 		s.Bitrate = state.Bitrate
	// 	}
	// }

	if state.Sid != "" {
		self.lock.Lock()
		for _, o := range self.observers {
			o.TaskStateNotify(t, state)
		}
		self.lock.Unlock()
	} else {
		clog.Infof("%s is over", t)
		self.DelTask(state.TaskId)
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
	tasks, err := LoadTasksByJobIdFromCache(jobid)
	if err != nil {
		clog.Errorf("load tasks by jobid(%s) error, %v", jobid, err)
		return
	}
	for _, t := range tasks {
		go t.Stop()
	}
}

func (self *TaskManager) Scheduler() TaskScheduler {
	return self.scheduler
}

func (self *TaskManager) SetCacheTimeout(second int) {
	self.cache_timeout = second
}

func SaveTaskToCache(t *Task, timeout int) (err error) {
	conn := cache.Get()
	defer conn.Close()
	key := fmt.Sprintf("task#%s", t.TaskId)
	_, err = conn.Do("HMSET", key, "task_id", t.TaskId, "job_id", t.JobId, "pid", t.Pid, "fid", t.Fid, "type", t.Type, "num_seg", t.NumSeg, "nid", t.Nid)
	if err != nil {
		return err
	}
	if _, err = conn.Do("EXPIRE", key, timeout); err != nil {
		return err
	}
	return nil
}

func UpdateTaskCache(task_id string, timeout int) error {
	conn := cache.Get()
	defer conn.Close()
	key := fmt.Sprintf("task#%s", task_id)
	if _, err := conn.Do("EXPIRE", key, timeout); err != nil {
		return err
	}
	return nil
}

func DelTaskCache(task_id string) error {
	conn := cache.Get()
	defer conn.Close()
	key := fmt.Sprintf("task#%s", task_id)
	if _, err := conn.Do("DEL", key); err != nil {
		return err
	}
	return nil
}

func LoadTaskFromCache(task_id string) (*Task, error) {
	conn := cache.Get()
	defer conn.Close()
	key := fmt.Sprintf("task#%s", task_id)
	t := new(Task)
	rsp, err := redis.Values(conn.Do("HGETALL", key))
	if err != nil {
		return nil, err
	}
	if err = redis.ScanStruct(rsp, t); err != nil {
		return nil, err
	}
	return t, nil
}

func LoadTasksByJobIdFromCache(job_id string) ([]*Task, error) {
	return LoadTasksFromCache(func(t *Task) bool {
		return t.JobId == job_id
	})
}

func LoadTasksByPidFromCache(pid string) ([]*Task, error) {
	return LoadTasksFromCache(func(t *Task) bool {
		return t.Pid == pid
	})
}

func LoadTasksFromCache(filter func(t *Task) bool) ([]*Task, error) {
	conn := cache.Get()
	defer conn.Close()
	keys, err := redis.Strings(conn.Do("KEYS", "task#*"))
	if err != nil {
		return nil, err
	}
	tasks := make([]*Task, 0, 10)
	for _, key := range keys {
		t, err := LoadTaskFromCache(key[5:])
		if err != nil {
			continue
		}
		if filter != nil {
			if !filter(t) {
				continue
			}
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
