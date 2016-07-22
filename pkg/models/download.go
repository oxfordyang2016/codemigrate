package models

import (
	"time"
)

type Download struct {
	Id       uint64            `xorm:"pk autoincr"`
	JobId    string            `xorm:"unique not null"`
	Pid      string            `xorm:"varchar(22) not null"`
	Uid      string            `xorm:"varchar(12) not null"`
	Finished bool              `xorm:"BOOL not null default(1)"`
	FinishAt time.Time         `xorm:"DateTime"`
	CreateAt time.Time         `xorm:"DateTime created"`
	UpdateAt time.Time         `xorm:"DateTime updated"`
	Details  []*DownloadDetail `xorm:"-"`
}

// 按照Uid查询下载的包,
// @param with_details 是否包含文件详细信息
// @param include_finished 是否包含已结束的包
func GetDownloadsByUid(uid string, with_details, include_finished bool) ([]*Download, error) {
	ua := make([]*Download, 0)
	var err error
	sess := DB().Where("uid=?", uid)
	if !include_finished {
		sess = sess.Where("finished=0")
	}
	if err = sess.Find(&ua); err != nil {
		return nil, err
	}
	if with_details {
		for _, u := range ua {
			if err = u.GetDetails(); err != nil {
				return nil, err
			}
		}
	}
	return ua, nil
}

// 按照Pid查询下载的包,
// @param with_details 是否包含文件详细信息
// @param include_finished 是否包含已结束的包
func GetDownloadsByPid(pid string, with_details, include_finished bool) ([]*Download, error) {
	ua := make([]*Download, 0)
	var err error
	sess := DB().Where("pid=?", pid)
	if !include_finished {
		sess = sess.Where("finished=0")
	}
	if err = sess.Find(&ua); err != nil {
		return nil, err
	}
	if with_details {
		for _, u := range ua {
			if err = u.GetDetails(); err != nil {
				return nil, err
			}
		}
	}
	return ua, nil
}

func (s *Download) TableName() string {
	return "package_download"
}

func (self *Download) GetDetails() error {
	var err error
	self.Details, err = GetDownloadDetails(self.JobId)
	if err == nil {
		for _, d := range self.Details {
			d.Job = self
		}
	}
	return err
}

type DownloadDetail struct {
	Id              uint64    `xorm:"pk autoincr"`
	JobId           string    `xorm:"not null"`
	Job             *Download `xorm:"-"`
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

func CreateDownloadDetail(job_id, fid string) (*DownloadDetail, error) {
	d := &DownloadDetail{
		JobId: job_id,
		Fid:   fid,
	}
	if _, err := DB().Insert(d); err != nil {
		return nil, err
	}
	return d, nil
}

func GetDownloadDetails(job_id string) ([]*DownloadDetail, error) {
	u := make([]*DownloadDetail, 0)
	if err := DB().Where("job_id=?", job_id).Find(&u); err != nil {
		return nil, err
	}
	return u, nil
}

func GetDownloadDetailsByFid(fid string) ([]*DownloadDetail, error) {
	u := make([]*DownloadDetail, 0)
	if err := DB().Where("fid=?", fid).Find(&u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *DownloadDetail) TableName() string {
	return "package_download_detail"
}
