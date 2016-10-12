package models

import (
	"./../../utils/db"
	"github.com/go-xorm/xorm"
)

var (
	Tables []interface{} = []interface{}{
		new(Node), new(Zone), new(Task),
	}
	Caches []interface{} = []interface{}{}
)

func DB() *xorm.Engine {
	return db.Engine
}

func SyncTables() error {
	return db.SyncTables(Tables)
}
