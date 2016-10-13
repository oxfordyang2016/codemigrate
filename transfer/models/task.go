package models

import (
	"time"
)

type Task struct {
	Id       int64     `xorm:"pk autoincr"`
	TaskId   string    `xorm:"varchar(255) not null unique"`
	Type     int       `xorm:"not null"`
	NodeId   string    `xorm:"varchar(255) not null"`
	ZoneId   string    `xorm:"varchar(255) not null"`
	JobId    string    `xorm:"varchar(255) not null"`
	Fid      string    `xorm:"varchar(255) not null"`
	NumSegs  int       `xorm:"not null"`
	CreateAt time.Time `xorm:"DateTime created"`
}

func CreateTask(t *Task) error {
	if t != nil {
		if _, err := DB().Insert(t); err != nil {
			return err
		}
	}
	return nil
}

func (self *Task) TableName() string {
	return "transfer_task"
}
