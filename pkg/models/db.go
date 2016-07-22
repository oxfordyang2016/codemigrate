package models

import (
	"./../../db"
	"github.com/go-xorm/xorm"
)

var (
	Models []interface{} = []interface{}{
		new(Pkg), new(File), new(Seg),
		new(Upload), new(UploadDetail),
		new(Download), new(DownloadDetail),
	}
)

func DB() *xorm.Engine {
	return db.DB()
}

func DBWrite() *xorm.Engine {
	return db.DBWrite()
}

func DBSyncTables() error {
	for _, m := range Models {
		if err := DB().Sync2(m); err != nil {
			return err
		}
	}
	return nil
}
