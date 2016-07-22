package models

import (
	"time"
)

type Pkg struct {
	Id             uint64 `xorm:"pk autoincr"`
	Pid            string `xorm:"varchar(22) unique not null"`
	Title          string `xorm:"varchar(30) not null"`
	Notes          string `xorm:"varchar(120)"`
	NumFiles       uint64 `xorm:"BigInt default(0) not null"`
	Size           uint64 `xorm:"BigInt default(0) not null"`
	BackupId       string `xorm:"varchar(32) unique"`
	EncryptionType int    `xorm:"Int default(0)"`
	// ServerPath     string    `xorm:"varchar(30) null"`
	Fmode     int       `xorm:"Int default(0)"`
	Flag      int       `xorm:"Int not null default(1)"`
	CreateAt  time.Time `xorm:"DateTime created"`
	UpdatedAt time.Time `xorm:"Datetime updated"`

	Files []*File `xorm:"-"`
}

func (s *Pkg) TableName() string {
	return "package_pkg"
}

// 创建Pkg
func CreatePkg(pid, title, notes string, num_files uint64, size uint64, encryption_type int) (*Pkg, error) {
	p := &Pkg{
		Pid:            pid,
		Title:          title,
		Notes:          notes,
		NumFiles:       num_files,
		Size:           size,
		EncryptionType: encryption_type,
	}
	if _, err := DB().Insert(p); err != nil {
		return nil, err
	}
	return p, nil
}

func GetPkg(uid, pid string) (*Pkg, error) {
	return nil, nil
}

func (self *Pkg) GetFiles(with_seg bool) error {
	var err error
	if self.Files, err = GetFiles(self.Pid, with_seg); err != nil {
		return err
	}
	for _, f := range self.Files {
		f.Pkg = self
	}
	return err
}
