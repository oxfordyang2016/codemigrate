package pkg

// job指每个用户下载的任务作为一个job或者自己的上传任务作为job.
// 用于以Pkg为单位的上传或者下载状态的跟踪和记录

import (
	"./../transfer/task"
	"./models"
	"cydex"
	"cydex/transfer"
	"fmt"
	clog "github.com/cihub/seelog"
	"sync"
	"time"
)

const (
	// JD数据同步进数据库的间隔
	// DEFAULT_CACHE_SYNC_TIMEOUT = 30 * time.Second
	DEFAULT_CACHE_SYNC_TIMEOUT = 0
	// 延时删除job的时间
	DELAY_DEL_JOB_TIME = 20 * time.Second
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
	return fmt.Sprintf("%s_%s_%s", uid, s, pid)
}

// 获取运行时segs的信息, 没有就添加进runtime
// size表示接收到的size, 不是seg的size
func getSegRuntime(jd *models.JobDetail, sid string) (s *models.Seg) {
	if jd.Segs == nil {
		clog.Trace("new segs map")
		jd.Segs = make(map[string]*models.Seg)
	}
	s, _ = jd.Segs[sid]
	if s == nil {
		s = new(models.Seg)
		s.Sid = sid
		jd.Segs[sid] = s
	}
	clog.Trace("seg num: ", len(jd.Segs))
	return
}

func updateJobDetail(jd *models.JobDetail, state *transfer.TaskState, seg_state int) (save bool) {
	jd.Bitrate = state.Bitrate
	if seg_state == cydex.TRANSFER_STATE_DONE {
		// NOTE: 因为f2tp下载时一个片段结束后得到的totalbytes是偏小的,所以使用数据库里的size来计算
		seg_m, _ := models.GetSeg(state.Sid)
		if seg_m != nil {
			state.RealTotalBytes = seg_m.Size
		}
		if jd.File == nil {
			jd.GetFile()
		}
		jd.CurSegSize = 0
		jd.FinishedSize += state.GetTotalBytes()
		jd.NumFinishedSegs++
		// cdxs-22, 保护数据不过限
		if jd.FinishedSize > jd.File.Size {
			jd.FinishedSize = jd.File.Size
		}
		if jd.NumFinishedSegs > jd.File.NumSegs {
			jd.NumFinishedSegs = jd.File.NumSegs
		}
		save = true
	} else {
		jd.CurSegSize = state.TotalBytes
		jd.State = seg_state
	}

	clog.Tracef("%s update: %d %d %d", jd, jd.NumFinishedSegs, jd.FinishedSize, jd.CurSegSize)

	// jd is finished?
	if jd.NumFinishedSegs == jd.File.NumSegs {
		clog.Infof("%s is finished", jd)
		jd.State = cydex.TRANSFER_STATE_DONE
		jd.FinishTime = time.Now()
		save = true
	}

	return
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

// type JobRuntime struct {
// 	*models.Job
// 	NumFinishedDetails int
// }

type JobObserver interface {
	OnJobCreate(*models.Job)
	OnJobStart(*models.Job)
	OnJobFinish(*models.Job)
}

type JobManager struct {
	lock               sync.Mutex
	cache_sync_timeout time.Duration //cache同步超时时间
	del_job_delay      time.Duration // 延迟删除job时间

	jobs          map[string]*models.Job // jobid->job, cache
	track_users   map[string]*Track      // uid->track, track里记录上传下载的pid
	track_pkgs    map[string]*Track      // pid->track, track里记录上传下载的uid
	track_deletes map[string]*Track      // uid->track, track里记录被删除的pid
	job_observers []JobObserver
}

func NewJobManager() *JobManager {
	jm := new(JobManager)
	jm.cache_sync_timeout = DEFAULT_CACHE_SYNC_TIMEOUT
	jm.jobs = make(map[string]*models.Job)
	jm.track_users = make(map[string]*Track)
	jm.track_pkgs = make(map[string]*Track)
	jm.track_deletes = make(map[string]*Track)
	jm.del_job_delay = DELAY_DEL_JOB_TIME
	return jm
}

// 创建一个新任务, 因为是活动任务,会加入cache
func (self *JobManager) CreateJob(uid, pid string, typ int) (err error) {
	clog.Infof("create job: u[%s], p[%s], t[%d]", uid, pid, typ)
	hashid := HashJob(uid, pid, typ)
	jobid := hashid

	// 已经存在的状态要复位,并重新加入track
	if job_m, _ := models.GetJob(jobid, false); job_m != nil {
		clog.Infof("%s is existed, add to track again", jobid)
		self.lock.Lock()
		defer self.lock.Unlock()

		// job需要reset, 重新开始
		job_m.GetDetails()
		for _, jd := range job_m.Details {
			jd.GetFile()
			if jd.File.Size > 0 {
				jd.Reset()
			} else {
				jd.StartTime = time.Now()
				jd.FinishTime = jd.StartTime
				jd.Save()
			}
		}

		// add track
		self.AddTrack(uid, pid, typ, false)
		// issue-1, 上传用户要监控下载用户状态,上传完的要加入track
		if typ == cydex.DOWNLOAD {
			upload_jobs, _ := models.GetJobsByPid(pid, cydex.UPLOAD, nil)
			for _, u_job := range upload_jobs {
				self.AddTrack(u_job.Uid, u_job.Pid, u_job.Type, false)
			}
		}

		// 删除原有的cache
		delete(self.jobs, jobid)

		// notify observers
		for _, o := range self.job_observers {
			o.OnJobCreate(job_m)
		}
		return nil
	}

	session := models.DB().NewSession()
	defer func() {
		models.SessionRelease(session)
		if err != nil {
			clog.Errorf("create job failed: %s", err)
		}
	}()
	if err = session.Begin(); err != nil {
		return
	}

	j := &models.Job{
		JobId: jobid,
		Uid:   uid,
		Pid:   pid,
		Type:  typ,
	}
	if _, err = session.Insert(j); err != nil {
		return err
	}
	clog.Debugf("insert a new Job: %s", jobid)
	// j.Details = make(map[string]*models.JobDetail)
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
		// jzh: 如果是0的文件或者文件夹,则状态就置为DONE, 因为客户端不会发送传输命令
		if f.Size == 0 {
			jd.StartTime = time.Now()
			jd.FinishTime = jd.StartTime
			jd.State = cydex.TRANSFER_STATE_DONE
		}
		if _, err := session.Insert(jd); err != nil {
			return err
		}
		// jd.Segs = make(map[string]*models.Seg)
		// j.Details[f.Fid] = jd
		clog.Tracef("insert job_detail fid:%s", f.Fid)
	}
	if err = session.Commit(); err != nil {
		return
	}

	self.lock.Lock()
	defer self.lock.Unlock()
	// add track
	self.AddTrack(uid, pid, typ, false)
	// issue-1, 上传用户要监控下载用户状态,上传完的要加入track
	if typ == cydex.DOWNLOAD {
		upload_jobs, _ := models.GetJobsByPid(pid, cydex.UPLOAD, nil)
		for _, u_job := range upload_jobs {
			self.AddTrack(u_job.Uid, u_job.Pid, u_job.Type, false)
		}
	}

	// notify observers
	for _, o := range self.job_observers {
		o.OnJobCreate(j)
	}

	return nil
}

// 删除job
func (self *JobManager) DeleteJob(uid, pid string, typ int) (err error) {
	hashid := HashJob(uid, pid, typ)
	clog.Infof("delete job: %s", hashid)
	job, _ := models.GetJob(hashid, false)
	if job == nil {
		return
	}
	if err = models.DeleteJob(hashid); err != nil {
		return
	}

	self.lock.Lock()
	defer self.lock.Unlock()

	self.DelTrack(uid, pid, typ, false)
	delete(self.jobs, hashid)
	// 要加入delete track, client通过增量接口获取
	self.AddTrackOfDelete(uid, pid, typ, false)

	clog.Debugf("delete job: %s over", hashid)

	return
}

// 从cache里取; 没有的话从数据库取; 如果是非finished,则加入cache
func (self *JobManager) GetJob(hashid string) *models.Job {
	var err error

	self.lock.Lock()
	defer self.lock.Unlock()

	j, _ := self.jobs[hashid]
	if j != nil {
		return j
	}

	if j, err = models.GetJob(hashid, true); err != nil {
		return nil
	}
	if j != nil {
		j.Details = make(map[string]*models.JobDetail)
		if !j.IsFinished() {
			j.GetDetails()
			j.NumUnfinishedDetails = j.CountUnfinishedDetails()
			// issue-6: 需要计数已经完成的jd
			// for _, jd := range j.Details {
			// 	if jd.State == cydex.TRANSFER_STATE_DONE {
			// 		j.NumFinishedDetails++
			// 	}
			// }
			// save to cache
			j.IsCached = true
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
		j.Details[fid] = jd
	}
	return
}

// implement task.TaskObserver
func (self *JobManager) AddTask(t *task.Task) {
	jobid := t.JobId
	job := self.GetJob(jobid)
	if job == nil {
		return
	}
	jd := self.GetJobDetail(jobid, t.Fid)
	if jd == nil {
		return
	}
	if jd.File == nil {
		jd.GetFile()
	}
	// issue-47, issue-50
	if jd.StartTime.IsZero() || (t.NumSegs == jd.File.NumSegs) {
		if jd.File.Size > 0 {
			jd.Reset()
			jd.StartTime = time.Now()
		} else {
			jd.StartTime = time.Now()
			jd.FinishTime = jd.StartTime
		}
		jd.Save()
	}

	if job.State == cydex.TRANSFER_STATE_IDLE {
		job.SaveState(cydex.TRANSFER_STATE_DOING)
		self.lock.Lock()
		defer self.lock.Unlock()
		for _, o := range self.job_observers {
			o.OnJobStart(job)
		}
	}
}

func (self *JobManager) DelTask(t *task.Task) {
	if t == nil {
		return
	}

	// cdxs-12: JobDetail和task状态要保持一致
	jobid := t.JobId
	j := self.GetJob(jobid)
	if j == nil {
		return
	}
	jd := self.GetJobDetail(jobid, t.Fid)
	if jd == nil {
		return
	}
	if jd.State == t.State {
		return
	}
	// NOTE: 有发生task超时，但客户端会重发请求并完成了传输，所以不允许超时task更改该状态
	if jd.State == cydex.TRANSFER_STATE_DONE {
		return
	}

	// NOTE: node的task会由于异常而停止，不一定带sid，所以jd也要更新。
	// 如果task是end状态，jd不一定是，例如边上传边下时task完了，但是jd不一定完毕。
	// end状态一般带sid，由TaskStateNotify来处理。如果不带，这边也没法处理，因为需要确认所有的segs完成才算完成
	if t.State == cydex.TRANSFER_STATE_PAUSE {
		jd.State = t.State
		jd.Save()
	}
}

func (self *JobManager) TaskStateNotify(t *task.Task, state *transfer.TaskState) {
	if t == nil || state == nil || state.Sid == "" {
		return
	}
	sid := state.Sid
	jobid := t.JobId
	j := self.GetJob(jobid)
	if j == nil {
		return
	}
	jd := self.GetJobDetail(jobid, t.Fid)
	if jd == nil {
		return
	}
	if jd.File == nil {
		jd.GetFile()
	}

	seg_state := t.State
	// 更新JobDetails状态, 根据判断更新Job状态, 是否finished?
	force_save := updateJobDetail(jd, state, seg_state)

	if jd.State == cydex.TRANSFER_STATE_DONE {
		j.NumUnfinishedDetails--
	}

	// // job is finished?
	// if j.NumFinishedDetails == len(j.Details) {
	// 	clog.Infof("%s is finished", j)
	// 	// j.SetState(cydex.TRANSFER_STATE_DONE)
	// 	self.lock.Lock()
	// 	delete(self.jobs, j.JobId)
	// 	self.DelTrack(j.Uid, j.Pid, j.Type, false)
	// 	self.lock.Unlock()
	// }

	//jzh: 将上传seg的发生变化的状态更新进数据库
	if j.Type == cydex.UPLOAD {
		seg_m, _ := models.GetSeg(sid)
		// clog.Tracef("sid:%s model_s:%d runtime_s:%d", sid, seg_m.State, seg_rt.State)
		if seg_m != nil && seg_m.State != seg_state {
			// clog.Trace(sid, "set state ", seg_rt.State)
			seg_m.SetState(seg_state)
		}
	}

	if force_save || time.Since(jd.UpdateAt) >= self.cache_sync_timeout {
		jd.Save()
	}

	self.ProcessJob(jobid)
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

	clog.Debugf("add track, u[%s], p[%s], t[%d]", uid, pid, typ)
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

// 删除track, issues-1, 上传用户要等下载完成后才能删除
func (self *JobManager) DelTrack(uid, pid string, typ int, mutex bool) {
	if mutex {
		self.lock.Lock()
		defer self.lock.Unlock()
	}

	if typ == cydex.UPLOAD {
		track, _ := self.track_pkgs[pid]
		if track != nil {
			// 如果还有下载则不退出
			if len(track.Downloads) > 0 {
				return
			}
		}
	}

	self.delTrack(uid, pid, typ)

	// 如果上传的pid, 无下载用户了,需要删除
	track, _ := self.track_pkgs[pid]
	if track != nil {
		if len(track.Downloads) == 0 {
			for uid, _ := range track.Uploads {
				self.delTrack(uid, pid, cydex.UPLOAD)
			}
		}
	}
}

func (self *JobManager) delTrack(uid, pid string, typ int) {
	clog.Debugf("del track, u[%s], p[%s], t[%d]", uid, pid, typ)
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
	clog.Debug("load tracks")
	jobs, err := models.GetUnFinishedJobs()
	if err != nil {
		return err
	}

	defer self.lock.Unlock()
	self.lock.Lock()
	self.ClearTracks(false)
	for _, j := range jobs {
		self.AddTrack(j.Uid, j.Pid, j.Type, false)
		// issue-1, 上传用户要监控下载用户状态,上传完的要加入track
		if j.Type == cydex.DOWNLOAD {
			upload_jobs, _ := models.GetJobsByPid(j.Pid, cydex.UPLOAD, nil)
			for _, u_job := range upload_jobs {
				self.AddTrack(u_job.Uid, u_job.Pid, u_job.Type, false)
			}
		}
	}
	return nil
}

// NOTE: cdxs-13, 为job增加了FinishedTimes字段，job需要同步一下状态
func (self *JobManager) JobsSyncState() error {
	clog.Infof("jobs sync state")
	jobs, err := models.GetJobs(cydex.DOWNLOAD, nil)
	if err == nil {
		for _, job := range jobs {
			if job.IsFinished() {
				if job.FinishedTimes == 0 {
					job.Finish()
				}
			}
		}
	}

	jobs, err = models.GetJobs(cydex.UPLOAD, nil)
	if err == nil {
		for _, job := range jobs {
			if job.IsFinished() {
				if job.FinishedTimes == 0 {
					job.Finish()
				}
			}
		}
	}

	return nil
}

// 增加delete_track的信息
func (self *JobManager) AddTrackOfDelete(uid string, pid string, typ int, mutex bool) {
	if mutex {
		self.lock.Lock()
		defer self.lock.Unlock()
	}

	clog.Debugf("add delete track, u[%s], p[%s], t[%d]", uid, pid, typ)
	track, _ := self.track_deletes[uid]
	if track == nil {
		track = NewTrack()
		self.track_deletes[uid] = track
	}
	if typ == cydex.UPLOAD {
		track.Uploads[pid] = 1
	} else {
		track.Downloads[pid] = 1
	}
}

// 获取delete_track的信息
// remove表示获取后是否删除,下次就取不到了
func (self *JobManager) getTrackOfDelete(uid string, typ int, remove bool) (pids []string) {
	var m map[string]int
	track, _ := self.track_deletes[uid]
	if track == nil {
		return
	}

	if typ == cydex.UPLOAD {
		m = track.Uploads
	} else {
		m = track.Downloads
	}
	for k, _ := range m {
		pids = append(pids, k)

		if remove {
			delete(m, k)
		}
	}

	if len(track.Uploads) == 0 && len(track.Downloads) == 0 {
		delete(self.track_deletes, uid)
	}

	return
}

func (self *JobManager) isJobFinished(job *models.Job) bool {
	if !job.IsCached {
		if job.CountUnfinishedDetails() == 0 {
			return true
		}
	} else {
		if job.NumUnfinishedDetails <= 0 {
			return true
		}
	}
	return false
}

func (self *JobManager) ProcessJob(jobid string) {
	job := self.GetJob(jobid)
	if job == nil {
		return
	}
	if self.isJobFinished(job) {
		clog.Infof("%s is finished", jobid)
		job.Finish()

		// 延时删除track和cache
		if self.del_job_delay > 0 {
			go func() {
				job := job
				time.Sleep(self.del_job_delay)
				self.lock.Lock()
				delete(self.jobs, jobid)
				self.DelTrack(job.Uid, job.Pid, job.Type, false)
				self.lock.Unlock()
			}()
		} else {
			self.lock.Lock()
			delete(self.jobs, jobid)
			self.DelTrack(job.Uid, job.Pid, job.Type, false)
			self.lock.Unlock()
		}

		self.lock.Lock()
		for _, o := range self.job_observers {
			o.OnJobFinish(job)
		}
		self.lock.Unlock()
	}
}

// 获取delete_track的信息
// remove表示获取后是否删除,下次就取不到了
func (self *JobManager) GetTrackOfDelete(uid string, typ int, remove, mutex bool) (pids []string) {
	if mutex {
		self.lock.Lock()
		defer self.lock.Unlock()
	}
	pids = self.getTrackOfDelete(uid, typ, remove)
	return
}

func (self *JobManager) GetUnFinishedJobCount() (upload, download int) {
	self.lock.Lock()
	defer self.lock.Unlock()

	for _, track := range self.track_pkgs {
		upload += len(track.Uploads)
		download += len(track.Downloads)
	}
	return
}

func (self *JobManager) ClearTracks(mutex bool) {
	if mutex {
		defer self.lock.Unlock()
		self.lock.Lock()
	}
	self.track_users = make(map[string]*Track)
	self.track_pkgs = make(map[string]*Track)
	self.track_deletes = make(map[string]*Track)
}

func (self *JobManager) AddJobObserver(o JobObserver) {
	if o == nil {
		return
	}
	defer self.lock.Unlock()
	self.lock.Lock()
	self.job_observers = append(self.job_observers, o)
}

// 更新JD进度
func UpdateJobDetailProcess(job_id, fid string, finished_size uint64, num_finished_segs int) {
	jd := JobMgr.GetJobDetail(job_id, fid)
	if jd == nil {
		return
	}
	clog.Tracef("%s update process: %d %d", jd, finished_size, num_finished_segs)
	jd.FinishedSize = finished_size
	jd.NumFinishedSegs = num_finished_segs
	jd.Save()
}
