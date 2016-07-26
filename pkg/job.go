package pkg

// job指每个用户下载的任务作为一个job或者自己的上传任务作为job.
// 用于以Pkg为单位的上传或者下载状态的跟踪和记录

import (
	"./../transfer/task"
	"./models"
	"cydex"
	"cydex/transfer"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	// JD数据同步进数据库的间隔
	DEFAULT_CACHE_SYNC_TIMEOUT = 30 * time.Second
)

var (
	JobMgr *JobManager
)

func init() {
	JobMgr = NewJobManager()
}

func GenerateHashId(uid, pid string, typ int) string {
	var s string
	switch typ {
	case cydex.UPLOAD:
		s = "U"
	case cydex.DOWNLOAD:
		s = "D"
	}
	return fmt.Sprintf("%s:%s:%s", uid, s, pid)
}

func updateJobDetailFinishedSize(jd *models.JobDetail, sid string, total_size uint64) uint64 {
	if jd.SegsRecvdSize == nil {
		jd.SegsRecvdSize = make(map[string]uint64)
	}
	jd.SegsRecvdSize[sid] = total_size
	total := uint64(0)
	for _, v := range jd.SegsRecvdSize {
		total += v
	}
	return total
}

func getHashIdFromTask(t *task.Task) string {
	var uid, pid string
	if t.UploadReq != nil {
		pid = t.UploadReq.Pid
		uid = t.UploadReq.Uid
	}
	if t.DownloadReq != nil {
		pid = t.DownloadReq.Pid
		uid = t.DownloadReq.Uid
	}
	return GenerateHashId(uid, pid, t.Type)
}

func getFidFromTask(t *task.Task) string {
	var fid string
	if t.UploadReq != nil {
		fid = t.UploadReq.Fid
	}
	if t.DownloadReq != nil {
		fid = t.DownloadReq.Fid
	}
	return fid
}

type JobManager struct {
	lock               sync.Mutex
	jobs               map[string]*models.Job // string is hashid
	cache_sync_timeout time.Duration          //cache同步超时时间
}

func NewJobManager() *JobManager {
	jm := new(JobManager)
	jm.jobs = make(map[string]*models.Job)
	jm.cache_sync_timeout = DEFAULT_CACHE_SYNC_TIMEOUT
	return jm
}

// 创建一个新任务, 因为是活动任务,会加入cache
func (self *JobManager) CreateJob(uid, pid string, typ int) (j *models.Job, err error) {
	session := models.DB().NewSession()
	session.Begin()

	defer func() {
		if err != nil {
			session.Rollback()
		}
		session.Close()
	}()

	hashid := GenerateHashId(uid, pid, typ)
	jobid := hashid
	j = &models.Job{
		JobId: jobid,
		Uid:   uid,
		Pid:   pid,
		Type:  typ,
	}
	if _, err = session.Insert(j); err != nil {
		return nil, err
	}
	j.Details = make(map[string]*models.JobDetail)
	// create details
	pkg, err := models.GetPkg(pid, true)
	if err != nil || pkg == nil {
		return nil, err
	}
	j.Pkg = pkg
	for _, f := range pkg.Files {
		jd := &models.JobDetail{
			JobId: jobid,
			Fid:   f.Fid,
		}
		if _, err := session.Insert(jd); err != nil {
			return nil, err
		}
		jd.SegsRecvdSize = make(map[string]uint64)
		j.Details[jd.Fid] = jd
	}
	session.Commit()

	self.lock.Lock()
	self.jobs[hashid] = j
	self.lock.Unlock()

	return j, nil
}

func (self *JobManager) getJob(hashid string) *models.Job {
	var err error

	self.lock.Lock()
	defer self.lock.Unlock()

	j, ok := self.jobs[hashid]
	if ok {
		return j
	}
	if j, err = models.GetJob(hashid, true); err != nil {
		return nil
	}
	if j != nil {
		// save to cache
		self.jobs[hashid] = j
	}
	return j
}

func (self *JobManager) getJobDetail(j *models.Job, fid string) *models.JobDetail {
	self.lock.Lock()
	defer self.lock.Unlock()

	if j.Details == nil {
		j.Details = make(map[string]*models.JobDetail)
	}
	jd, ok := j.Details[fid]
	if ok {
		return jd
	}
	jd = j.GetDetail(fid)
	if jd == nil {
		return nil
	}
	j.Details[fid] = jd
	return jd
}

// implement task.TaskObserver
func (self *JobManager) AddTask(t *task.Task) {
	hashid := getHashIdFromTask(t)
	j := self.getJob(hashid)
	if j == nil {
		return
	}
	jd := self.getJobDetail(j, getFidFromTask(t))
	if jd == nil {
		return
	}
	if jd.StartTime.IsZero() {
		jd.SetStartTime(time.Now())
	}

	// jzh:不清楚是续传还是补传还是重新下载, 不好处理, api协议有缺陷
	// // TODO: 要处理已经finished的,然后重新下载的
	// if t.DownlaodReq != nil && t.DownloadReq.FinishedSidList != nil {
	// 	//TODO 断点续传, 需要重新计算FinishedSize和NumFinishedSeg
	// }
}

func (self *JobManager) DelTask(t *task.Task) {

}

func (self *JobManager) TaskStateNotify(t *task.Task, state *transfer.TaskState) {
	hashid := getHashIdFromTask(t)
	j := self.getJob(hashid)
	if j == nil {
		return
	}
	jd := self.getJobDetail(j, getFidFromTask(t))
	if jd == nil {
		return
	}

	// 更新JobDetails状态, 根据判断更新Job状态, 是否finished?
	jd.FinishedSize = updateJobDetailFinishedSize(jd, state.Sid, state.TotalBytes)
	jd.Bitrate = state.Bitrate
	update := true

	s := strings.ToLower(state.State)
	switch s {
	case "transferring":
		jd.State = cydex.TRANSFER_STATE_DOING
	case "interrupt":
		jd.State = cydex.TRANSFER_STATE_PAUSE
	case "end":
		if jd.File == nil {
			jd.GetFile()
		}
		jd.NumFinishedSegs++
		if jd.File.NumSegs == jd.NumFinishedSegs {
			jd.Finish()
			update = false
			j.NumFinishedDetails++
		}
	}

	if j.NumFinishedDetails == int(j.Pkg.NumFiles) {
		j.Finish()
		self.lock.Lock()
		delete(self.jobs, j.JobId) // 从表里删除
		self.lock.Unlock()
	}

	if update && time.Since(jd.UpdateAt) >= self.cache_sync_timeout {
		jd.Save()
	}
}

func (self *JobManager) SetCacheSyncTimeout(d time.Duration) {
	self.cache_sync_timeout = d
}

func (self *JobManager) HasCachedJob(jobid string) bool {
	defer self.lock.Unlock()
	self.lock.Lock()
	_, ok := self.jobs[jobid]
	return ok
}
