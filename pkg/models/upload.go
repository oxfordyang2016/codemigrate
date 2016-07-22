package models

import (
	"time"
)

type Upload struct {
	Id       uint64          `xorm:"pk autoincr"`
	JobId    string          `xorm:"unique not null"`
	Pid      string          `xorm:"varchar(22) not null"`
	Uid      string          `xorm:"varchar(12) not null"`
	Finished bool            `xorm:"BOOL not null default(0)"`
	FinishAt time.Time       `xorm:"DateTime"`
	CreateAt time.Time       `xorm:"DateTime created"`
	UpdateAt time.Time       `xorm:"DateTime updated"`
	Details  []*UploadDetail `xorm:"-"`
}

// 按照Uid查询上传的包,
// @param with_details 是否包含文件详细信息
// @param include_finished 是否包含已结束的包
func GetUploadsByUid(uid string, with_details, include_finished bool) ([]*Upload, error) {
	ua := make([]*Upload, 0)
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

// 按照Pid查询上传的包,
// @param with_details 是否包含文件详细信息
// @param include_finished 是否包含已结束的包
func GetUploadsByPid(pid string, with_details, include_finished bool) ([]*Upload, error) {
	ua := make([]*Upload, 0)
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

func GetUpload(pid, uid string, with_details bool) (*Upload, error) {
	u := new(Upload)
	existed, err := DB().Where("pid=? and uid=?", pid, uid).Get(u)
	if !existed || err != nil {
		return nil, err
	}
	if with_details {
		if err = u.GetDetails(); err != nil {
			return nil, err
		}
	}
	return u, nil
}

func (self *Upload) TableName() string {
	return "package_upload"
}

func (self *Upload) GetDetails() error {
	var err error
	self.Details, err = GetUploadDetails(self.JobId)
	if err == nil {
		for _, d := range self.Details {
			d.Job = self
		}
	}
	return err
}

// File-User上传详细表
type UploadDetail struct {
	Id                uint64    `xorm:"pk autoincr"`
	JobId             string    `xorm:"not null"`
	Job               *Upload   `xorm:"-"`
	Fid               string    `xorm:"varchar(24) not null"`
	File              *File     `xorm:"-"`
	StartTime         time.Time `xorm:"DateTime"`
	FinishTime        time.Time `xorm:"DateTime"`
	FinishedSize      uint64    `xorm:"BigInt not null default(0)"`
	State             int       `xorm:"Int not null default(0)"`
	NumTransferedSegs int       `xorm:"not null"`
	Checked           int       `xorm:"Int default(0)"`
	CreateAt          time.Time `xorm:"DateTime created"`
	UpdatedAt         time.Time `xorm:"Datetime updated"`
}

func GetUploadDetails(job_id string) ([]*UploadDetail, error) {
	u := make([]*UploadDetail, 0)
	if err := DB().Where("job_id=?", job_id).Find(&u); err != nil {
		return nil, err
	}
	return u, nil
}

func GetUploadDetailsByFid(fid string) ([]*UploadDetail, error) {
	u := make([]*UploadDetail, 0)
	if err := DB().Where("fid=?", fid).Find(&u); err != nil {
		return nil, err
	}
	return u, nil
}

func (self *UploadDetail) TableName() string {
	return "package_upload_detail"
}
