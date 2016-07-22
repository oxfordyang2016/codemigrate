package models

import (
	"time"
)

type Job struct {
	Id       uint64       `xorm:"pk autoincr"`
	JobId    string       `xorm:"unique not null"`
	Type     int          `xorm:"int"` // cydex.UPLOAD or cydex.DOWNLOAD
	Pid      string       `xorm:"varchar(22) not null"`
	Uid      string       `xorm:"varchar(12) not null"`
	Finished bool         `xorm:"BOOL not null default(0)"`
	FinishAt time.Time    `xorm:"DateTime"`
	CreateAt time.Time    `xorm:"DateTime created"`
	UpdateAt time.Time    `xorm:"DateTime updated"`
	Details  []*JobDetail `xorm:"-"`
}

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
	if details {
		for _, j := range jobs {
			if err = j.GetDetails(); err != nil {
				return nil, err
			}
		}
	}
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

func (self *Job) GetDetails() error {
	var err error
	self.Details, err = GetDetails(self.JobId)
	if err == nil {
		for _, d := range self.Details {
			d.Job = self
		}
	}
	return err
}

type JobDetail struct {
	Id              uint64    `xorm:"pk autoincr"`
	JobId           string    `xorm:"not null"`
	Job             *Job      `xorm:"-"`
	Fid             string    `xorm:"varchar(24) not null"`
	File            *File     `xorm:"-"`
	StartTime       time.Time `xorm:"DateTime"`
	Finishtime      time.Time `xorm:"DateTime"`
	FinishedSize    uint64    `xorm:"BigInt not null default(0)"`
	State           int       `xorm:"Int not null default(0)"`
	NumFinishedSegs int       `xorm:"not null default(0)"`
	Checked         int       `xorm:"Int not null default(0)"`
	CreateAt        time.Time `xorm:"DateTime created"`
	UpdatedAt       time.Time `xorm:"DateTime updated"`
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

func GetJobDetails(job_id string) ([]*JobDetail, error) {
	ds := make([]*JobDetail, 0)
	if err := DB().Where("job_id=?", job_id).Find(&ds); err != nil {
		return nil, err
	}
	return ds, nil
}

func GetDownloadDetailsByFid(fid string) ([]*JobDetail, error) {
	ds := make([]*JobDetail, 0)
	if err := DB().Where("fid=?", fid).Find(&ds); err != nil {
		return nil, err
	}
	return ds, nil
}

func (self *JobDetail) TableName() string {
	return "package_job_detail"
}
