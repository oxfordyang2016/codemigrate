package models

import (
	"./../../utils/db"
	"github.com/go-xorm/xorm"
)

var (
	Tables []interface{} = []interface{}{
		new(Pkg), new(File), new(Seg),
		new(Job), new(JobDetail),
	}
)

func DB() *xorm.Engine {
	return db.Engine
}

func SyncTables() error {
	return db.SyncTables(Tables)
}

func SessionRelease(sess *xorm.Session) {
	if !sess.IsCommitedOrRollbacked {
		sess.Rollback()
	}
	sess.Close()
}
