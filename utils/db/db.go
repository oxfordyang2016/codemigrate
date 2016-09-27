package db

import (
	"errors"
	_ "github.com/go-sql-driver/mysql"
	// "github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	_ "github.com/mattn/go-sqlite3"
)

var (
	Engine *xorm.Engine
	Cache  *xorm.LRUCacher
)

const (
	CACHE_ITEM_NUM = 2000
)

func CreateEngine(driver, url string, show_sql bool) error {
	var err error
	if Engine, err = xorm.NewEngine(driver, url); err != nil {
		return err
	}
	if CACHE_ITEM_NUM > 0 {
		Cache = xorm.NewLRUCacher(xorm.NewMemoryStore(), CACHE_ITEM_NUM)
		// Engine.SetDefaultCacher(cacher)
	}
	// Engine.Logger().SetLevel(core.LOG_DEBUG)
	Engine.ShowSQL(show_sql)
	return nil
}

func SetupEngine(e *xorm.Engine) error {
	if Engine != nil {
		return errors.New("Engine has been setup")
	}
	Engine = e
	return nil
}

func SyncTables(tables []interface{}) error {
	for _, t := range tables {
		if err := Engine.Sync2(t); err != nil {
			return err
		}
	}
	return nil
}

func MapCache(tables []interface{}) {
	if Cache == nil {
		return
	}
	for _, t := range tables {
		Engine.MapCacher(t, Cache)
	}
}
