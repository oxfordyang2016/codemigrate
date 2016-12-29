package models

import (
	"time"
)

type UserProfile struct {
	Id           int64     `xorm:"pk autoincr"`
	Uid          string    `xorm:"varchar(12) not null"`
	Ulevel       int       `xorm:"not null"`
	LastLogin    time.Time `xorm:"DateTime"`
	BlackList    bool      `xorm:"BOOL not null 'blacklist'"`
	BlackTime    time.Time `xorm:"DateTime"`
	UserPath     string    `xorm:"varchar(30)"`
	UserSpace    int64     `xorm:"BIGINT"`
	UsedSpace    int64     `xorm:"BIGINT"`
	GroupId      int64     `xorm:"null"`
	UserId       int64     `xorm:"unique not null"` // jzh: UserId是AuthUser.Id的外键
	User         *AuthUser `xorm:"-"`
	Language     int       `xorm:"not null"`
	EmailAffairs uint      `xorm:"not null"`
}

func (self *UserProfile) TableName() string {
	return "package_userprofile"
}

func GetUserProfile(uid string) (up *UserProfile, err error) {
	up = new(UserProfile)
	var existed bool
	if existed, err = DB().Where("uid=?", uid).Get(up); err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	up.User, err = GetAuthUser(up.UserId)
	if err != nil {
		return nil, err
	}
	return up, nil
}
