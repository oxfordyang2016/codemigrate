package models

import (
	"time"
)

type Transfer struct {
	Id           int64     `xorm:"pk autoincr"`
	NodeId       string    `xorm:"varchar(255) notnull unique"`
	RxTotalBytes uint64    `xorm:"default(0)"`
	RxMaxBitrate uint64    `xorm:"default(0)"`
	RxMinBitrate uint64    `xorm:"default(0)"`
	RxAvgBitrate uint64    `xorm:"default(0)"`
	RxTasks      uint64    `xorm:"default(0)"`
	TxTotalBytes uint64    `xorm:"default(0)"`
	TxMaxBitrate uint64    `xorm:"default(0)"`
	TxMinBitrate uint64    `xorm:"default(0)"`
	TxAvgBitrate uint64    `xorm:"default(0)"`
	TxTasks      uint64    `xorm:"default(0)"`
	CreateAt     time.Time `xorm:"DateTime created"`
	UpdateAt     time.Time `xorm:"Datetime updated"`
}

func GetTransfer(nodeid string) (t *Transfer, err error) {
	t = new(Transfer)
	var existed bool
	if existed, err = DB().Where("node_id=?", nodeid).Get(t); err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	return t, nil
}

func GetTransfers() (transfers []*Transfer, err error) {
	transfers = make([]*Transfer, 0)
	if err := DB().Find(&transfers); err != nil {
		return nil, err
	}
	return
}

func NewTransfer(t *Transfer) error {
	if _, err := DB().Insert(t); err != nil {
		return err
	}
	return nil
}

func NewTransferWithId(nodeid string) (*Transfer, error) {
	o := &Transfer{
		NodeId: nodeid,
	}
	if _, err := DB().Insert(o); err != nil {
		return nil, err
	}
	return o, nil
}

func (self *Transfer) Update() error {
	_, err := DB().Id(self.Id).AllCols().Update(self)
	return err
}

func (self *Transfer) TableName() string {
	return "stat_transfer"
}
