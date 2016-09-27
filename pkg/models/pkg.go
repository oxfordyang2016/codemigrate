package models

import (
	"cydex"
	"time"
)

type Pkg struct {
	Id             int64  `xorm:"pk autoincr"`
	Pid            string `xorm:"varchar(22) unique not null"`
	Title          string `xorm:"varchar(30) not null"`
	Notes          string `xorm:"varchar(120)"`
	NumFiles       uint64 `xorm:"BigInt default(0) not null"`
	Size           uint64 `xorm:"BigInt default(0) not null"`
	BackupId       string `xorm:"varchar(32)"`
	EncryptionType int    `xorm:"Int default(0)"`
	// ServerPath     string    `xorm:"varchar(30) null"`
	Fmode    int             `xorm:"Int default(0)"`
	Flag     int             `xorm:"Int not null default(1)"`
	MetaData *cydex.MetaData `xorm:"TEXT json"`
	CreateAt time.Time       `xorm:"DateTime created"`
	UpdateAt time.Time       `xorm:"Datetime updated"`

	Files []*File `xorm:"-"`
}

func (s *Pkg) TableName() string {
	return "package_pkg"
}

// 创建Pkg
func CreatePkg(p *Pkg) error {
	if _, err := DB().Insert(p); err != nil {
		return err
	}
	return nil
}

// func CreatePkg(pid, title, notes string, num_files uint64, size uint64, encryption_type int) (*Pkg, error) {
// 	p := &Pkg{
// 		Pid:            pid,
// 		Title:          title,
// 		Notes:          notes,
// 		NumFiles:       num_files,
// 		Size:           size,
// 		EncryptionType: encryption_type,
// 	}
// 	if _, err := DB().Insert(p); err != nil {
// 		return nil, err
// 	}
// 	return p, nil
// }

func GetPkgs() (pkgs []*Pkg, err error) {
	pkgs = make([]*Pkg, 0)
	if err := DB().Find(&pkgs); err != nil {
		return nil, err
	}
	return
}

func GetPkg(pid string, with_files bool) (p *Pkg, err error) {
	p = new(Pkg)
	var existed bool
	if existed, err = DB().Where("pid=?", pid).Get(p); err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	p.GetFiles(false)
	return p, nil
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
