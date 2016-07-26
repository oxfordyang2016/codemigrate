package models

import (
	"./../../db"
	"github.com/go-xorm/xorm"
)

var (
	Tables []interface{} = []interface{}{
		new(Node), new(Zone),
	}
)

func DB() *xorm.Engine {
	return db.Engine
}

func SyncTables() error {
	return db.SyncTables(Tables)
}
