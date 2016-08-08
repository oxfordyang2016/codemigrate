package models

import (
	"cydex"
	"errors"
	"fmt"
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
		err = j.GetPkg(false)
	}
	return j, err
}

func GetJobsByUid(uid string, typ int, p *cydex.Pagination) ([]*Job, error) {
	jobs := make([]*Job, 0)
	var err error
	sess := DB().Where("uid=? and type=?", uid, typ)
	if p != nil {
		sess = sess.Limit(p.PageSize, (p.PageNum-1)*p.PageSize)
	}
	if err = sess.Find(&jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func GetJobsByPid(pid string, typ int) ([]*Job, error) {
	jobs := make([]*Job, 0)
	var err error
	sess := DB().Where("pid=? and type=?", pid, typ)
	if err = sess.Find(&jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

// 查询未完成的任务
func GetUnFinishedJobs() ([]*Job, error) {
	jobs := make([]*Job, 0)
	var err error
	if err = DB().Where("finished=0").Find(&jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

// 删除job
func DeleteJob(jobid string) (err error) {
	j := &Job{JobId: jobid}
	has, err := DB().Get(j)
	if err != nil {
		return err
	}
	if !has {
		return nil
	}

	session := DB().NewSession()
	defer SessionRelease(session)
	if err := session.Begin(); err != nil {
		return err
	}
	jds := make([]*JobDetail, 0, 100)
	if err = session.Where("job_id=?", jobid).Find(&jds); err != nil {
		return err
	}
	for _, jd := range jds {
		if _, err = session.Delete(jd); err != nil {
			return err
		}
	}
	session.Delete(j)
	if err = session.Commit(); err != nil {
		return err
	}

	return nil
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

func (self *Job) GetDetails() error {
	jds, err := GetJobDetails(self.JobId)
	if err != nil {
		return err
	}
	if self.Details == nil {
		self.Details = make(map[string]*JobDetail)
	}
	for _, jd := range jds {
		self.Details[jd.Fid] = jd
	}
	return err
}

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

func (self *Job) GetPkg(with_files bool) (err error) {
	if self.Pkg, err = GetPkg(self.Pid, with_files); err != nil {
		return
	}
	return
}

func (self *Job) String() string {
	return fmt.Sprintf("<Job(%s)", self.JobId)
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
	Bitrate uint64          `xorm:"-"`
	Segs    map[string]*Seg `xorm:"-"` //sid->seg
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

func (self *JobDetail) String() string {
	return fmt.Sprintf("<JobDetail(%s:%s)", self.JobId, self.Fid)
}
