package models

import (
	"time"
)

// 区域, 多个Node在一个Zone,共享同一存储空间
type Zone struct {
	Id       uint64    `xorm:"pk autoincr"`
	Zid      string    `xorm:"varchar(255) not null unique"`
	Name     string    `xorm:"varchar(255) not null"`
	CreateAt time.Time `xorm:"DateTime created"`
	UpdateAt time.Time `xorm:"DateTime updated"`
	Nodes    []*Node   `xorm:"-"`
}

func CreateZone(zid, name string) error {
	z := &Zone{
		Zid:  zid,
		Name: name,
	}
	if _, err := DB().Insert(z); err != nil {
		return err
	}
	return nil
}

func GetZone(zid string) (z *Zone, err error) {
	z = new(Zone)
	var existed bool
	if existed, err = DB().Where("zid=?", zid).Get(z); err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	return z, nil
}
