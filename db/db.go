package db

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	_ "github.com/mattn/go-sqlite3"
)

var (
	default_db *DBEngine
)

type DBEngine struct {
	Read  *xorm.Engine
	Write *xorm.Engine
}

func CreateDefaultDBEngine(driver, name string, debug bool) error {
	var err error
	default_db, err = NewDBEngine(driver, name, debug)
	return err
}

func DB() *xorm.Engine {
	return default_db.Read
}

func DBWrite() *xorm.Engine {
	return default_db.Write
}

func NewDBEngine(driver, name string, debug bool) (*DBEngine, error) {
	engine, err := xorm.NewEngine(driver, name)
	if err != nil {
		return nil, err
	}
	engine.ShowSQL(debug)

	d := &DBEngine{
		Read:  engine,
		Write: engine,
	}
	return d, nil
}

func (self *DBEngine) SyncModels(models []interface{}) error {
	for _, m := range models {
		if err := self.Write.Sync2(m); err != nil {
			return err
		}
	}
	return nil
}
