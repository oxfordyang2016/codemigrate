package db

import (
	_ "github.com/go-sql-driver/mysql"
	// "github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	_ "github.com/mattn/go-sqlite3"
)

var (
	UserEngine *xorm.Engine
	// Cache      *xorm.LRUCacher
)

func CreateUserEngine(driver, url string, show_sql bool) error {
	var err error
	if UserEngine, err = xorm.NewEngine(driver, url); err != nil {
		return err
	}
	// if CACHE_ITEM_NUM > 0 {
	// 	Cache = xorm.NewLRUCacher(xorm.NewMemoryStore(), CACHE_ITEM_NUM)
	// 	// Engine.SetDefaultCacher(cacher)
	// }
	// Engine.Logger().SetLevel(core.LOG_DEBUG)
	UserEngine.ShowSQL(show_sql)
	return nil
}

func SyncUserTables(tables []interface{}) error {
	for _, t := range tables {
		if err := UserEngine.Sync2(t); err != nil {
			return err
		}
	}
	return nil
}
