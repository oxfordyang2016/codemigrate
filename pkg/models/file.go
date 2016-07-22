package models

import (
	"time"
)

type File struct {
	Id         uint64    `xorm:"pk autoincr"`
	Fid        string    `xorm:"varchar(24) notnull unique"`
	Name       string    `xorm:"varchar(30) notnull"`
	Size       uint64    `xorm:"BigInt notnull"`
	EigenValue string    `xorm:"varchar(80)"`
	Path       string    `xorm:"varchar(128) notnull"`
	PathAbs    string    `xorm:"varchar(128)"`
	Pid        string    `xorm:"varchar(22)"`
	Type       int       `xorm:"Int notnull default(1)"`
	ServerPath string    `xorm:"varchar(30)"`
	CreateAt   time.Time `xorm:"DateTime created"`
	UpdatedAt  time.Time `xorm:"Datetime updated"`
	NumSegs    int       `xorm:"not null"`
	Pkg        *Pkg      `xorm:"-"`
	Segs       []*Seg    `xorm:"-"`
}

// 创建文件
func CreateFile(fid, name, path string, size uint64, num_segs int) (*File, error) {
	f := &File{
		Fid:     fid,
		Name:    name,
		Path:    path,
		Size:    size,
		NumSegs: num_segs,
	}
	if _, err := DB().Insert(f); err != nil {
		return nil, err
	}
	return f, nil
}

func GetFiles(pid string, with_seg bool) ([]*File, error) {
	files := make([]*File, 0)
	if err := DB().Where("pid=?", pid).Find(&files); err != nil {
		return nil, err
	}
	if with_seg {
		//TODO
		for _, f := range files {
			f = f
		}
	}
	return files, nil
}

func (self *File) TableName() string {
	return "package_file"
}

func (self *File) GetSegs() (err error) {
	if self.Segs, err = GetSegs(self.Fid); err != nil {
		return
	}
	for _, s := range self.Segs {
		s.File = self
	}
	return
}

// func (self *File) QueryUploads() ([]*UploadDetail, error) {
// 	return GetUploadDetailsByFid(self.Fid)
// }
//
// func (self *File) QueryDownloads() ([]*DownloadDetail, error) {
// 	return GetDownloadDetailsByFid(self.Fid)
// }
