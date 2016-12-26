package notify

import (
	// "./../pkg"
	pkg_model "./../pkg/models"
	"./../utils"
	"./../utils/cache"
	"cydex"
	"encoding/json"
	clog "github.com/cihub/seelog"
	"github.com/garyburd/redigo/redis"
	"time"
)

/* 收到JobObserver通知后，进入pool处理，获取详细信息，包括用户信息，生成PkgEvent, 通过redis.pubsub通知外部模块进行发送
 */

const (
	MaxEventProcesser = 50 // 处理Event的最大执行数
	MaxTryTimes       = 3  // 最大Publish次数
	PubSubChanForPkg  = "cydex.event.pkg"
)

var (
	NotifyEvents = []int{
		cydex.NotifyToReceiverNewPkg,
		cydex.NotifyToSenderPkgUploadFinish,
		cydex.NotifyToReceiverPkgDownloadFinish,
		cydex.NotifyToSenderPkgDownloadStart,
		cydex.NotifyToSenderPkgDownloadFinish,
	}
)

type JobInfo struct {
	JobId    string
	Type     int
	Pid      string
	Uid      string
	CreateAt time.Time
	UpdateAt time.Time
	SoftDel  int
}

type PkgInfo struct {
	Pid            string
	Title          string
	Notes          string
	NumFiles       uint64
	Size           uint64
	HumanSize      string
	EncryptionType int
	Fmode          int
	Flag           int
	MetaData       *cydex.MetaData
	CreateAt       time.Time
	UpdateAt       time.Time
}

// 包裹事件
type PkgEvent struct {
	// 包裹的名称，事件，发送者，接收者等, 供Handler处理
	NotifyType int
	Time       time.Time
	Job        *JobInfo
	Pkg        *PkgInfo
	User       *cydex.User
	Owner      *cydex.User
}

type NotifyManage struct {
	Enable           bool
	max_ev_processer int
	pkg_ev_queue     chan *PkgEvent
	max_try_times    int // 通知最大发送次数
}

func NewNotifyManage(max_ev_processer, max_try_times int) *NotifyManage {
	o := new(NotifyManage)
	o.pkg_ev_queue = make(chan *PkgEvent)

	if max_ev_processer <= 0 {
		max_ev_processer = MaxEventProcesser
	}
	o.max_ev_processer = max_ev_processer

	if max_try_times <= 0 {
		max_try_times = MaxTryTimes
	}
	o.max_try_times = max_try_times
	o.SetEnable(false) //default is false

	return o
}

func (self *NotifyManage) SetEnable(v bool) {
	self.Enable = v
}

// implement pkg.JobObserver
func (self *NotifyManage) OnJobCreate(job *pkg_model.Job) {
	if !self.Enable {
		return
	}

	if job.Type == cydex.UPLOAD {
		return
	}

	pkg_ev := &PkgEvent{
		NotifyType: cydex.NotifyToReceiverNewPkg,
		Time:       time.Now(),
		Job: &JobInfo{
			JobId:    job.JobId,
			Type:     job.Type,
			Pid:      job.Pid,
			Uid:      job.Uid,
			CreateAt: job.CreateAt,
			UpdateAt: job.UpdateAt,
		},
	}

	self.pkg_ev_queue <- pkg_ev
}

func (self *NotifyManage) OnJobStart(job *pkg_model.Job) {
	if !self.Enable {
		return
	}

	if job.Type == cydex.UPLOAD {
		return
	}

	// 下载job 和 NotifyToSenderPkgDownloadStart
	pkg_ev := &PkgEvent{
		NotifyType: cydex.NotifyToSenderPkgDownloadStart,
		Time:       time.Now(),
		Job: &JobInfo{
			JobId:    job.JobId,
			Type:     job.Type,
			Pid:      job.Pid,
			Uid:      job.Uid,
			CreateAt: job.CreateAt,
			UpdateAt: job.UpdateAt,
		},
	}

	self.pkg_ev_queue <- pkg_ev
}

func (self *NotifyManage) OnJobFinish(job *pkg_model.Job) {
	if !self.Enable {
		return
	}

	// 自己传输完毕
	var notify_type int
	switch job.Type {
	case cydex.UPLOAD:
		notify_type = cydex.NotifyToSenderPkgUploadFinish
	case cydex.DOWNLOAD:
		notify_type = cydex.NotifyToReceiverPkgDownloadFinish
	default:
		return
	}

	pkg_ev := &PkgEvent{
		NotifyType: notify_type,
		Time:       time.Now(),
		Job: &JobInfo{
			JobId:    job.JobId,
			Type:     job.Type,
			Pid:      job.Pid,
			Uid:      job.Uid,
			CreateAt: job.CreateAt,
			UpdateAt: job.UpdateAt,
		},
	}

	self.pkg_ev_queue <- pkg_ev

	clog.Debugf("on job finish, %s", job)
	// 通知发送者某用户接收完毕
	if job.Type == cydex.DOWNLOAD {
		pkg_ev := &PkgEvent{
			NotifyType: cydex.NotifyToSenderPkgDownloadFinish,
			Time:       time.Now(),
			Job: &JobInfo{
				JobId:    job.JobId,
				Type:     job.Type,
				Pid:      job.Pid,
				Uid:      job.Uid,
				CreateAt: job.CreateAt,
				UpdateAt: job.UpdateAt,
			},
		}

		self.pkg_ev_queue <- pkg_ev
	}
}

// End of implement pkg.JobObserver

func (self *NotifyManage) Serve() {
	sem := make(chan int, self.max_ev_processer)
	for ev := range self.pkg_ev_queue {
		select {
		case sem <- 1:
		default:
			clog.Warnf("[notify] publish is too busy!")
			continue
		}
		ev := ev
		go func() {
			self.publishPkgEvent(ev)
			<-sem
		}()
	}
}

func (self *NotifyManage) fetchJobInfo(pkg_ev *PkgEvent) error {
	pid := pkg_ev.Job.Pid
	pkg_m, err := pkg_model.GetPkg(pid, false)
	if err != nil {
		return err
	}
	pkg_ev.Pkg = &PkgInfo{
		Pid:            pkg_m.Pid,
		Title:          pkg_m.Title,
		Notes:          pkg_m.Notes,
		NumFiles:       pkg_m.NumFiles,
		Size:           pkg_m.Size,
		EncryptionType: pkg_m.EncryptionType,
		Fmode:          pkg_m.Fmode,
		Flag:           pkg_m.Flag,
		MetaData:       pkg_m.MetaData,
		CreateAt:       pkg_m.CreateAt,
		UpdateAt:       pkg_m.UpdateAt,
	}
	pkg_ev.Pkg.HumanSize = utils.GetHumanSize(pkg_ev.Pkg.Size)

	// TODO get user by job.Uid

	if pkg_ev.Job.Type == cydex.UPLOAD {
		pkg_ev.Owner = pkg_ev.User
	} else {
		upload_jobs, err := pkg_model.GetJobsByPid(pid, cydex.UPLOAD, nil)
		if err != nil || upload_jobs == nil {
			return err
		}
		upload_job := upload_jobs[0]
		// TODO get owner by upload_job.Uid
		upload_job = upload_job
	}
	return nil
}

func (self *NotifyManage) publishPkgEvent(pkg_ev *PkgEvent) {
	clog.Debug("publish pkg event")
	if err := self.fetchJobInfo(pkg_ev); err != nil {
		clog.Errorf("[notify] fetch job info failed, err:%s", err.Error())
		return
	}

	// 发送到redis.pubsub
	b, err := json.Marshal(pkg_ev)
	if err != nil {
		clog.Errorf("[notify] marshal pkg_ev failed, err:%s", err.Error())
		return
	}

	for i := 0; i < self.max_try_times; i++ {
		_, err = redis.Int(cache.Get().Do("PUBLISH", PubSubChanForPkg, string(b)))
		if err != nil {
			clog.Errorf("[notify] publish pkg event failed: %s, times:%d", err.Error(), i+1)
			continue
		}
		clog.Tracef("[notify] publish pkg event ok")
		break
	}
}
