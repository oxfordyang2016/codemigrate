package task

import (
	trans "./.."
	"cydex"
	"cydex/transfer"
	"errors"
	"fmt"
	"github.com/pborman/uuid"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	TASK_TIMEOUT = 5 * time.Minute
)

var (
	TaskMgr *TaskManager
	Logger  *log.Logger
)

func init() {
	Logger = log.New(os.Stderr, "task", log.LstdFlags|log.Lshortfile)
	TaskMgr = NewTaskManager()
}

func SetLogger(l *log.Logger) {
	if l != nil {
		Logger = l
	}
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
	Pid string
}

func NewUploadReq(r *transfer.UploadTaskReq, pid string) (req *UploadReq, err error) {
	if r == nil || pid == "" {
		return nil, errors.New("Invalid params")
	}
	req = &UploadReq{
		UploadTaskReq: r,
		Pid:           pid,
	}
	return req, nil
}

type DownloadReq struct {
	*transfer.DownloadTaskReq
	Pid             string
	FinishedSidList []string //已经下完的片段
}

func NewDownloadReq(r *transfer.DownloadTaskReq, pid string, finished_sid_list []string) (req *DownloadReq, err error) {
	if r == nil || pid == "" {
		return nil, errors.New("Invalid params")
	}
	req = &DownloadReq{
		DownloadTaskReq: r,
		Pid:             pid,
		FinishedSidList: finished_sid_list,
	}
	return req, nil
}

type Task struct {
	TaskId      string             // 任务ID
	Type        int                // 类型, U or D
	State       transfer.TaskState // 任务状态
	Node        *trans.Node        // 该任务分配的传输节点
	UploadReq   *UploadReq
	DownloadReq *DownloadReq
	Fid         string
	SidList     []string
	CreateAt    time.Time
	UpdateAt    time.Time
}

func (self *Task) IsDispatched() bool {
	return self.Node != nil
}

func (self *Task) String() string {
	var s string
	switch self.Type {
	case cydex.UPLOAD:
		s = "u"
	case cydex.DOWNLOAD:
		s = "d"
	default:
		s = "?"
	}
	return fmt.Sprintf("<Task(%s %s)>", self.TaskId, s)
}

// 任务管理器
type TaskManager struct {
	scheduler              TaskScheduler
	lock                   sync.Mutex
	tasks                  map[string]*Task // task_id -> node
	observer_idx           uint32
	observers              map[uint32]TaskObserver
	task_state_routine_run bool
	task_state_chan        chan *transfer.TaskState
}

func NewTaskManager() *TaskManager {
	t := new(TaskManager)
	t.tasks = make(map[string]*Task)
	t.observers = make(map[uint32]TaskObserver)
	t.task_state_chan = make(chan *transfer.TaskState, 10)
	return t
}

func (self *TaskManager) SetScheduler(s TaskScheduler) {
	Logger.Printf("Set scheduler as %s\n", s.String())
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
	for _, t := range self.tasks {
		if time.Since(t.UpdateAt) > TASK_TIMEOUT {
			self.DelTask(t.TaskId)
		}
	}

	defer func() {
		self.lock.Unlock()
		Logger.Printf("Add task %s\n", t)
	}()

	self.lock.Lock()

	self.tasks[t.TaskId] = t
	for _, o := range self.observers {
		o.AddTask(t)
	}
}

func (self *TaskManager) GetTask(taskid string) (t *Task) {
	defer self.lock.Unlock()
	self.lock.Lock()
	t, _ = self.tasks[taskid]
	return
}

func (self *TaskManager) DelTask(taskid string) {
	defer self.lock.Unlock()
	self.lock.Lock()
	t, _ := self.tasks[taskid]
	delete(self.tasks, taskid)
	if t != nil {
		for _, o := range self.observers {
			o.DelTask(t)
		}
	}
	Logger.Printf("Del task(%s) %+v\n", taskid, t)
}

func (self *TaskManager) DispatchUpload(req *UploadReq, timeout time.Duration) (rsp *transfer.UploadTaskRsp, node *trans.Node, err error) {
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
		// Log
		return
	}

	var rsp_msg *transfer.Message
	msg := transfer.NewReqMessage("", "uploadtask", "", 0)
	msg.Req.UploadTask = req.UploadTaskReq
	if rsp_msg, err = node.SendRequestSync(msg, timeout); err != nil {
		// log
		return
	}
	// TODO 判断rsp_msg的Code值
	rsp = rsp_msg.Rsp.UploadTask

	t := &Task{
		TaskId:    req.TaskId,
		Type:      cydex.UPLOAD,
		Node:      node,
		UploadReq: req,
		CreateAt:  time.Now(),
		UpdateAt:  time.Now(),
		Fid:       req.UploadTaskReq.Fid,
		SidList:   req.UploadTaskReq.SidList,
	}
	TaskMgr.AddTask(t)
	return
}

func (self *TaskManager) DispatchDownload(req *DownloadReq, timeout time.Duration) (rsp *transfer.DownloadTaskRsp, node *trans.Node, err error) {
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

	var rsp_msg *transfer.Message
	msg := transfer.NewReqMessage("", "downloadtask", "", 0)
	msg.Req.DownloadTask = req.DownloadTaskReq
	if rsp_msg, err = node.SendRequestSync(msg, timeout); err != nil {
		// log
		return
	}
	rsp = rsp_msg.Rsp.DownloadTask

	t := &Task{
		TaskId:      req.TaskId,
		Type:        cydex.DOWNLOAD,
		Node:        node,
		DownloadReq: req,
		CreateAt:    time.Now(),
		UpdateAt:    time.Now(),
		Fid:         req.DownloadTaskReq.Fid,
		SidList:     req.DownloadTaskReq.SidList,
	}
	TaskMgr.AddTask(t)
	return
}

func (self *TaskManager) OnTaskStateNotify(nid string, state *transfer.TaskState) (err error) {
	if !self.task_state_routine_run {
		self.task_state_routine_run = true
		go self.taskStateThread()
	}
	self.task_state_chan <- state
	return
}

func (self *TaskManager) handleTaskState(state *transfer.TaskState) (err error) {
	Logger.Printf("TaskState: %+v\n", state)
	t := self.GetTask(state.TaskId)
	if t != nil {
		t.UpdateAt = time.Now()
	}

	self.lock.Lock()
	for _, o := range self.observers {
		o.TaskStateNotify(t, state)
	}
	self.lock.Unlock()

	// task有多个seg, 只有这些seg均end了,才触发DelTask~~~
	if state.Sid == "" {
		// sid为空说明是针对整个task的
		s := strings.ToLower(state.State)
		if s == "end" {
			self.DelTask(state.TaskId)
		}
	}
	return
}

func (self *TaskManager) taskStateThread() {
	for state := range self.task_state_chan {
		self.handleTaskState(state)
	}
}
