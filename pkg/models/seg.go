package models

import (
	"time"
)

type Seg struct {
	Id    uint64 `xorm:"pk autoincr"`
	Sid   string `xorm:"varchar(32) unique not null"`
	State int    `xorm:"Int not null default(0)"`
	Size  uint64 `xorm:"BigInt not null"`
	// Storage  string    `xorm:"varchar(255)"`
	Fid      string    `xorm:"varchar(24)"`
	CreateAt time.Time `xorm:"DateTime created"`
	UpdateAt time.Time `xorm:"Datetime updated"`
	File     *File     `xorm:"-"`
}

// 创建Seg
func CreateSeg(sid, fid string, size uint64) (*Seg, error) {
	s := &Seg{
		Sid:  sid,
		Fid:  fid,
		Size: size,
	}
	if _, err := DB().Insert(s); err != nil {
		return nil, err
	}
	return s, nil
}

func GetSeg(sid string) (s *Seg, err error) {
	s = new(Seg)
	var existed bool
	if existed, err = DB().Where("sid=?", sid).Get(s); err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	return s, nil
}

func GetSegs(fid string) ([]*Seg, error) {
	sa := make([]*Seg, 0)
	if err := DB().Where("fid=?", fid).Find(&sa); err != nil {
		return nil, err
	}
	return sa, nil
}

func (self *Seg) SetState(state int) error {
	s := &Seg{
		State: state,
	}
	_, err := DB().Id(self.Id).Cols("state").Update(s)
	if err == nil {
		self.State = state
	}
	return err
}

// func (self *Seg) SetStorage(storage string) error {
// 	s := &Seg{
// 		Storage: storage,
// 	}
// 	_, err := DB().Id(self.Id).Cols("storage").Update(s)
// 	if err == nil {
// 		self.Storage = storage
// 	}
// 	return err
// }

func (s *Seg) TableName() string {
	return "package_seg"
}
