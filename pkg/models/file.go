package models

import (
	"time"
)

type File struct {
	Id         uint64    `xorm:"pk autoincr"`
	Fid        string    `xorm:"varchar(24) notnull unique"`
	Pid        string    `xorm:"varchar(22)"`
	Name       string    `xorm:"varchar(255) notnull"`
	Path       string    `xorm:"TEXT notnull"`
	Type       int       `xorm:"Int notnull default(1)"`
	Size       uint64    `xorm:"BigInt notnull"`
	Mode       int       `xorm:'Int default(0)'`
	EigenValue string    `xorm:"varchar(255)"`
	PathAbs    string    `xorm:"TEXT"`
	NumSegs    int       `xorm:"not null"`
	Storage    string    `xorm:"varchar(255)"`
	CreateAt   time.Time `xorm:"DateTime created"`
	UpdatedAt  time.Time `xorm:"Datetime updated"`
	Pkg        *Pkg      `xorm:"-"`
	Segs       []*Seg    `xorm:"-"`
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
		for _, f := range files {
			if err := f.GetSegs(); err != nil {
				return nil, err
			}
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

func (self *File) SetStorage(storage string) error {
	self.Storage = storage
	_, err := DB().Id(self.Id).Cols("storage").Update(self)
	return err
}
