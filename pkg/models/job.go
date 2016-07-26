package models

import (
	"cydex"
	"errors"
	"time"
)

type Job struct {
	Id       uint64    `xorm:"pk autoincr"`
	JobId    string    `xorm:"unique not null"`
	Type     int       `xorm:"int"` // cydex.UPLOAD or cydex.DOWNLOAD
	Pid      string    `xorm:"varchar(22) not null"`
	Pkg      *Pkg      `xorm:"-"`
	Uid      string    `xorm:"varchar(12) not null"`
	Finished bool      `xorm:"BOOL not null default(0)"`
	FinishAt time.Time `xorm:"DateTime"`
	CreateAt time.Time `xorm:"DateTime created"`
	UpdateAt time.Time `xorm:"DateTime updated"`

	// runtime usage
	Details            map[string]*JobDetail `xorm:"-"`
	NumFinishedDetails int                   `xorm:"-"`
}

func CreateJob(jobid, uid, pid string, typ int) (*Job, error) {
	j := &Job{
		JobId: jobid,
		Uid:   uid,
		Pid:   pid,
		Type:  typ,
	}
	if _, err := DB().Insert(j); err != nil {
		return nil, err
	}
	return j, nil
}

func GetJob(jobid string, with_pkg bool) (*Job, error) {
	j := new(Job)
	existed, err := DB().Where("job_id=?", jobid).Get(j)
	if err != nil || !existed {
		return nil, nil
	}
	if with_pkg {
		j.Pkg, err = GetPkg(j.Pid, false)
	}
	return j, err
}

// // 按照(uid, pid, type)获取一个job对象
// func GetActiveJobByUidAndPid(uid, pid string, typ int, details bool) (*Job, error) {
// 	j := new(Job)
// 	var existed bool
// 	if existed, err = DB().Where("uid=? and pid=? and type=? and finished=0", uid, pid, typ).Get(j); err != nil {
// 		return nil, nil
// 	}
// 	if !existed {
// 		return nil, nil
// 	}
// 	return j, nil
// }

// 按照Uid和type查询job
// @param with_details 是否包含详细信息
// @param include_finished 是否包含已结束的job
func GetJobsByUid(uid string, typ int, details, finished bool) ([]*Job, error) {
	jobs := make([]*Job, 0)
	var err error
	sess := DB().Where("uid=? and type=?", uid, typ)
	if !finished {
		sess = sess.Where("finished=0")
	}
	if err = sess.Find(&jobs); err != nil {
		return nil, err
	}
	// if details {
	// 	for _, j := range jobs {
	// 		if err = j.GetDetails(); err != nil {
	// 			return nil, err
	// 		}
	// 	}
	// }
	return jobs, nil
}

// // 按照Pid查询下载的包,
// // @param with_details 是否包含文件详细信息
// // @param include_finished 是否包含已结束的包
// func GetDownloadsByPid(pid string, with_details, include_finished bool) ([]*Download, error) {
// 	ua := make([]*Download, 0)
// 	var err error
// 	sess := DB().Where("pid=?", pid)
// 	if !include_finished {
// 		sess = sess.Where("finished=0")
// 	}
// 	if err = sess.Find(&ua); err != nil {
// 		return nil, err
// 	}
// 	if with_details {
// 		for _, u := range ua {
// 			if err = u.GetDetails(); err != nil {
// 				return nil, err
// 			}
// 		}
// 	}
// 	return ua, nil
// }

func (self *Job) TableName() string {
	return "package_job"
}

// func (self *Job) GetDetails() error {
// 	ds, err := GetJobDetails(self.JobId)
// 	if err == nil {
// 		if self.Details == nil {
// 			self.Details = make(map[string]*JobDetail)
// 		}
// 		for _, d := range ds {
// 			self.
// 		}
// 	}
// 	return err
// }

func (self *Job) Finish() error {
	j := &Job{
		Finished: true,
		FinishAt: time.Now(),
	}
	_, err := DB().Where("job_id=?", self.JobId).Cols("finished", "finish_at").Update(j)
	if err == nil {
		self.Finished = j.Finished
		self.FinishAt = j.FinishAt
	}
	return err
}

func (self *Job) GetDetail(fid string) *JobDetail {
	if self.Details == nil {
		self.Details = make(map[string]*JobDetail)
	}
	jd, err := GetJobDetail(self.JobId, fid)
	if err != nil {
		return nil
	}
	return jd
}

type JobDetail struct {
	Id              uint64    `xorm:"pk autoincr"`
	JobId           string    `xorm:"not null"`
	Job             *Job      `xorm:"-"`
	Fid             string    `xorm:"varchar(24) not null"`
	File            *File     `xorm:"-"`
	StartTime       time.Time `xorm:"DateTime"`
	FinishTime      time.Time `xorm:"DateTime"`
	FinishedSize    uint64    `xorm:"BigInt not null default(0)"`
	State           int       `xorm:"Int not null default(0)"`
	NumFinishedSegs int       `xorm:"not null default(0)"`
	Checked         int       `xorm:"Int not null default(0)"`
	CreateAt        time.Time `xorm:"DateTime created"`
	UpdateAt        time.Time `xorm:"DateTime updated"`

	// runtime
	Bitrate       uint64            `xorm:"-"`
	SegsRecvdSize map[string]uint64 `xorm:"-"` //sid->size
}

// 批量创建
func CreateJobDetails(jds []*JobDetail) error {
	if jds == nil {
		return errors.New("job details slice is nil")
	}
	_, err := DB().Insert(&jds)
	return err
}

func CreateJobDetail(job_id, fid string) (*JobDetail, error) {
	d := &JobDetail{
		JobId: job_id,
		Fid:   fid,
	}
	if _, err := DB().Insert(d); err != nil {
		return nil, err
	}
	return d, nil
}

func GetJobDetail(jobid, fid string) (jd *JobDetail, err error) {
	jd = new(JobDetail)
	var existed bool
	if existed, err = DB().Where("job_id=? and fid=?", jobid, fid).Get(jd); err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	return jd, nil
}

func GetJobDetails(job_id string) ([]*JobDetail, error) {
	ds := make([]*JobDetail, 0)
	if err := DB().Where("job_id=?", job_id).Find(&ds); err != nil {
		return nil, err
	}
	return ds, nil
}

func GetJobDetailsByFid(fid string) ([]*JobDetail, error) {
	ds := make([]*JobDetail, 0)
	if err := DB().Where("fid=?", fid).Find(&ds); err != nil {
		return nil, err
	}
	return ds, nil
}

func (self *JobDetail) GetFile() (err error) {
	var f *File
	if f, err = GetFile(self.Fid); err != nil {
		return err
	}
	self.File = f
	return
}

func (self *JobDetail) SetStartTime(t time.Time) error {
	if t.IsZero() {
		t = time.Now()
	}
	jd := &JobDetail{
		StartTime: t,
	}
	_, err := DB().Id(self.Id).Update(jd)
	if err == nil {
		self.StartTime = t
	}
	return err
}

func (self *JobDetail) SetState(state int) error {
	jd := &JobDetail{
		State: state,
	}
	_, err := DB().Id(self.Id).Cols("state").Update(jd)
	if err == nil {
		self.State = state
	}
	return err
}

func (self *JobDetail) Finish() error {
	jd := &JobDetail{
		State:      cydex.TRANSFER_STATE_DONE,
		FinishTime: time.Now(),
	}
	_, err := DB().Id(self.Id).Cols("state", "finish_time").Update(jd)
	if err == nil {
		self.State = jd.State
		self.FinishTime = jd.FinishTime
	}
	return err
}

// 全保存
func (self *JobDetail) Save() error {
	_, err := DB().Id(self.Id).AllCols().Update(self)
	return err
}

func (self *JobDetail) TableName() string {
	return "package_job_detail"
}
