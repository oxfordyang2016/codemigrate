package main

import (
	_ "./api/"
	"./db"
	"./pkg"
	pkg_model "./pkg/models"
	trans_model "./transfer/models"
	"github.com/astaxie/beego"
)

func initDB() (err error) {
	// 创建数据库
	if err = db.CreateEngine("sqlite3", "/tmp/cydex.sqlite3", false); err != nil {
		return
	}
	// 同步表结构
	if err = pkg_model.SyncTables(); err != nil {
		return
	}
	if err = trans_model.SyncTables(); err != nil {
		return
	}
	return
}

func start() {
	initDB()
	// 设置拆包器
	pkg.SetUnpacker(pkg.NewDefaultUnpacker(50*1024*1024, 25))
}

func main() {
	start()
	beego.Run()
}
