package models

import (
	"time"
)

type AuthUser struct {
	Id          int64     `xorm:"pk autoincr"`
	Password    string    `xorm:"varchar(128) not null"`
	LastLogin   time.Time `xorm:"DateTime not null"`
	IsSuperuser bool      `xorm:"BOOL not null"`
	Username    string    `xorm:"varchar(64) unique not null"`
	FirstName   string    `xorm:"varchar(30) not null"`
	LastName    string    `xorm:"varchar(30) not null"`
	Email       string    `xorm:"varchar(254) not null"`
	IsStaff     bool      `xorm:"BOOL not null"`
	IsActive    bool      `xorm:"BOOL not null"`
	DateJoined  time.Time `xorm:"DateTime not null"`
}

func (self *AuthUser) TableName() string {
	return "auth_user"
}

func GetAuthUser(id int64) (au *AuthUser, err error) {
	au = new(AuthUser)
	var existed bool
	if existed, err = DB().Id(id).Get(au); err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	return
}
