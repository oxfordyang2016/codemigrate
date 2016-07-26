package models

import (
	"time"
)

type File struct {
	Id         uint64 `xorm:"pk autoincr"`
	Fid        string `xorm:"varchar(24) notnull unique"`
	Pid        string `xorm:"varchar(22)"`
	Name       string `xorm:"varchar(30) notnull"`
	Path       string `xorm:"varchar(128) notnull"`
	Type       int    `xorm:"Int notnull default(1)"`
	Size       uint64 `xorm:"BigInt notnull"`
	EigenValue string `xorm:"varchar(80)"`
	PathAbs    string `xorm:"varchar(128)"`
	NumSegs    int    `xorm:"not null"`
	// ServerPath string    `xorm:"varchar(30)"`
	CreateAt  time.Time `xorm:"DateTime created"`
	UpdatedAt time.Time `xorm:"Datetime updated"`
	Pkg       *Pkg      `xorm:"-"`
	Segs      []*Seg    `xorm:"-"`
}

// 创建文件
func CreateFile(fid, pid, name, path string, size uint64, num_segs int) (*File, error) {
	f := &File{
		Fid:     fid,
		Name:    name,
		Path:    path,
		Size:    size,
		Pid:     pid,
		NumSegs: num_segs,
	}
	if _, err := DB().Insert(f); err != nil {
		return nil, err
	}
	return f, nil
}

func GetFile(fid string) (f *File, err error) {
	f = new(File)
	var existed bool
	if existed, err = DB().Where("fid=?", fid).Get(f); err != nil {
		return nil, err
	}
	if !existed {
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
