package models

import (
	"./../../utils/db"
	"github.com/go-xorm/xorm"
)

var (
	Tables []interface{} = []interface{}{
		new(AuthUser), new(UserProfile),
	}
	Caches []interface{} = []interface{}{}
)

func DB() *xorm.Engine {
	return db.UserEngine
}
