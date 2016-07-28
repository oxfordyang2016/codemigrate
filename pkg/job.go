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

func HashJob(uid, pid string, typ int) string {
	var s string
	switch typ {
	case cydex.UPLOAD:
		s = "U"
	case cydex.DOWNLOAD:
		s = "D"
	}
	return fmt.Sprintf("%s:%s:%s", uid, s, pid)
}

func getMetaFromTask(t *task.Task) (uid, pid, fid string, typ int) {
	if t == nil {
		return
	}
	typ = t.Type
	if t.UploadReq != nil {
		pid = t.UploadReq.Pid
		uid = t.UploadReq.Uid
		fid = t.UploadReq.Fid
	}
	if t.DownloadReq != nil {
		pid = t.DownloadReq.Pid
		uid = t.DownloadReq.Uid
		fid = t.DownloadReq.Fid
	}
	return
}

func getHashIdFromTask(t *task.Task) string {
	uid, pid, _, typ := getMetaFromTask(t)
	return HashJob(uid, pid, typ)
}

func getFidFromTask(t *task.Task) (fid string) {
	_, _, fid, _ = getMetaFromTask(t)
	return
}

// 获取运行时segs的信息, 没有就添加进runtime
func getSegRuntime(jd *models.JobDetail, sid string) (s *models.Seg) {
	if jd.Segs == nil {
		jd.Segs = make(map[string]*models.Seg)
	}
	s, _ = jd.Segs[sid]
	if s == nil {
		s = new(models.Seg)
		s.Sid = sid
		jd.Segs[sid] = s
	}
	return
}

func updateJobDetail(jd *models.JobDetail) {
	// finished size
	total := uint64(0)
	for _, s := range jd.Segs {
		total += s.Size
	}
	jd.FinishedSize = total

	// state
	for _, s := range jd.Segs {
		if s.State == cydex.TRANSFER_STATE_DOING {
			jd.State = cydex.TRANSFER_STATE_DOING
			break
		}
		if s.State == cydex.TRANSFER_STATE_PAUSE {
			jd.State = cydex.TRANSFER_STATE_PAUSE
			break
		}
	}
}

type Track struct {
	// int没啥用,这里当作set使用
	Uploads   map[string]int
	Downloads map[string]int
}

func NewTrack() *Track {
	t := new(Track)
	t.Uploads = make(map[string]int)
	t.Downloads = make(map[string]int)
	return t
}

type JobRuntime struct {
	*models.Job
	NumFinishedDetails int
}

type JobManager struct {
	lock               sync.Mutex
	cache_sync_timeout time.Duration //cache同步超时时间

	jobs        map[string]*models.Job // jobid
	track_users map[string]*Track      // uid->track, track里记录上传下载的pid
	track_pkgs  map[string]*Track      // pid->track, track里记录上传下载的uid
}

func NewJobManager() *JobManager {
	jm := new(JobManager)
	jm.cache_sync_timeout = DEFAULT_CACHE_SYNC_TIMEOUT
	jm.jobs = make(map[string]*models.Job)
	jm.track_users = make(map[string]*Track)
	jm.track_pkgs = make(map[string]*Track)
	return jm
}

// 创建一个新任务, 因为是活动任务,会加入cache
func (self *JobManager) CreateJob(uid, pid string, typ int) (err error) {
	session := models.DB().NewSession()
	session.Begin()

	defer func() {
		if err != nil {
			session.Rollback()
		}
		session.Close()
	}()

	hashid := HashJob(uid, pid, typ)
	jobid := hashid
	j := &models.Job{
		JobId: jobid,
		Uid:   uid,
		Pid:   pid,
		Type:  typ,
	}
	if _, err = session.Insert(j); err != nil {
		return err
	}
	j.Details = make(map[string]*models.JobDetail)
	// create details
	pkg, err := models.GetPkg(pid, true)
	if err != nil || pkg == nil {
		return err
	}
	j.Pkg = pkg
	for _, f := range pkg.Files {
		jd := &models.JobDetail{
			JobId: jobid,
			Fid:   f.Fid,
		}
		if _, err := session.Insert(jd); err != nil {
			return err
		}
		jd.Segs = make(map[string]*models.Seg)
		j.Details[f.Fid] = jd
	}
	session.Commit()

	self.lock.Lock()
	self.jobs[hashid] = j
	self.AddTrack(uid, pid, typ, false)
	self.lock.Unlock()

	return nil
}

// 从cache里取; 没有的话从数据库取; 如果是非finished,则加入cache
func (self *JobManager) GetJob(hashid string) *models.Job {
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
		if !j.Finished {
			j.GetDetails()
			// save to cache
			self.jobs[hashid] = j
		}
	}
	return j
}

// 从cache里取,没有则从数据库取
func (self *JobManager) GetJobDetail(jobid, fid string) (jd *models.JobDetail) {
	var err error
	j := self.GetJob(jobid)
	if j == nil {
		return
	}
	var ok bool
	jd, ok = j.Details[fid]
	if !ok {
		if jd, err = models.GetJobDetail(jobid, fid); err != nil {
			return nil
		}
	}
	return
}

// implement task.TaskObserver
func (self *JobManager) AddTask(t *task.Task) {
	hashid := getHashIdFromTask(t)
	jd := self.GetJobDetail(hashid, getFidFromTask(t))
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
	uid, pid, _, typ := getMetaFromTask(t)
	sid := state.Sid
	if sid == "" || uid == "" || pid == "" {
		return
	}
	hashid := getHashIdFromTask(t)
	j := self.GetJob(hashid)
	if j == nil {
		return
	}
	jd := self.GetJobDetail(hashid, getFidFromTask(t))
	if jd == nil {
		return
	}
	if jd.File == nil {
		jd.GetFile()
	}
	seg_rt := getSegRuntime(jd, sid)

	// 更新JobDetails状态, 根据判断更新Job状态, 是否finished?
	seg_rt.Size = state.TotalBytes
	jd.Bitrate = state.Bitrate
	update := true

	s := strings.ToLower(state.State)
	switch s {
	case "transferring":
		seg_rt.State = cydex.TRANSFER_STATE_DOING
	case "interrupt":
		seg_rt.State = cydex.TRANSFER_STATE_PAUSE
	case "end":
		seg_rt.State = cydex.TRANSFER_STATE_DONE
		jd.NumFinishedSegs++
	default:
		return
	}

	updateJobDetail(jd)

	// jd is finished
	if jd.NumFinishedSegs == jd.File.NumSegs {
		jd.Finish()
		update = false
		j.NumFinishedDetails++
	}

	if j.NumFinishedDetails == int(j.Pkg.NumFiles) {
		j.Finish()
		self.lock.Lock()
		delete(self.jobs, j.JobId) // 从表里删除
		self.DelTrack(uid, pid, typ, false)
		self.lock.Unlock()
	}

	//jzh: 将上传seg的发生变化的状态更新进数据库
	if typ == cydex.UPLOAD {
		seg_m, _ := models.GetSeg(sid)
		if seg_m != nil && seg_m.State != seg_rt.State {
			seg_m.SetState(seg_rt.State)
		}
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

func (self *JobManager) AddTrack(uid, pid string, typ int, mutex bool) {
	if mutex {
		self.lock.Lock()
		defer self.lock.Unlock()
	}

	track, _ := self.track_pkgs[pid]
	if track == nil {
		track = NewTrack()
		self.track_pkgs[pid] = track
	}
	if typ == cydex.UPLOAD {
		track.Uploads[uid] = 1
	} else {
		track.Downloads[uid] = 1
	}

	track, _ = self.track_users[uid]
	if track == nil {
		track = NewTrack()
		self.track_users[uid] = track
	}
	if typ == cydex.UPLOAD {
		track.Uploads[pid] = 1
	} else {
		track.Downloads[pid] = 1
	}
}

func (self *JobManager) DelTrack(uid, pid string, typ int, mutex bool) {
	if mutex {
		self.lock.Lock()
		defer self.lock.Unlock()
	}

	track, _ := self.track_pkgs[pid]
	if track != nil {
		if typ == cydex.UPLOAD {
			delete(track.Uploads, uid)
		} else {
			delete(track.Downloads, uid)
		}
		if len(track.Uploads) == 0 && len(track.Downloads) == 0 {
			delete(self.track_pkgs, pid)
		}
	}

	track, _ = self.track_users[uid]
	if track != nil {
		if typ == cydex.UPLOAD {
			delete(track.Uploads, pid)
		} else {
			delete(track.Downloads, pid)
		}
		if len(track.Uploads) == 0 && len(track.Downloads) == 0 {
			delete(self.track_users, uid)
		}
	}
}

func (self *JobManager) GetPkgTrack(pid string, typ int) (uids []string) {
	self.lock.Lock()
	defer self.lock.Unlock()

	var m map[string]int
	track, _ := self.track_pkgs[pid]
	if track != nil {
		if typ == cydex.UPLOAD {
			m = track.Uploads
		} else {
			m = track.Downloads
		}
		for k, _ := range m {
			uids = append(uids, k)
		}
	}
	return
}

func (self *JobManager) GetUserTrack(uid string, typ int) (pids []string) {
	self.lock.Lock()
	defer self.lock.Unlock()

	var m map[string]int
	track, _ := self.track_users[uid]
	if track != nil {
		if typ == cydex.UPLOAD {
			m = track.Uploads
		} else {
			m = track.Downloads
		}
		for k, _ := range m {
			pids = append(pids, k)
		}
	}
	return
}

// 从cache中获取jobs信息
func (self *JobManager) GetJobsByUid(uid string, typ int) (jobs []*models.Job, err error) {
	pids := self.GetUserTrack(uid, typ)
	for _, pid := range pids {
		hashid := HashJob(uid, pid, typ)
		job := self.GetJob(hashid)
		if job != nil {
			jobs = append(jobs, job)
		}
	}
	return
}

// 从cache中获取jobs信息
func (self *JobManager) GetJobsByPid(pid string, typ int) (jobs []*models.Job, err error) {
	uids := self.GetPkgTrack(pid, typ)
	for _, uid := range uids {
		hashid := HashJob(uid, pid, typ)
		job := self.GetJob(hashid)
		if job != nil {
			jobs = append(jobs, job)
		}
	}
	return
}

// 从数据库中同步track信息
func (self *JobManager) LoadTracks() error {
	jobs, err := models.GetUnFinishedJobs()
	if err != nil {
		return err
	}

	defer self.lock.Unlock()
	self.lock.Lock()
	self.ClearTracks(false)
	for _, j := range jobs {
		self.AddTrack(j.Uid, j.Pid, j.Type, false)
	}
	return nil
}

func (self *JobManager) ClearTracks(mutex bool) {
	if mutex {
		defer self.lock.Unlock()
		self.lock.Lock()
	}
	self.track_users = make(map[string]*Track)
	self.track_pkgs = make(map[string]*Track)
}
