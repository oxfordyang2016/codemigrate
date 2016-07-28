package main

import (
	_ "./api/"
	"./db"
	"./pkg"
	pkg_model "./pkg/models"
	trans_model "./transfer/models"
	"github.com/astaxie/beego"
	clog "github.com/cihub/seelog"
)

func initLog() {
	cfgfiles := []string{"seelog.xml", "/opt/cydex/seelog.xml"}
	for _, file := range cfgfiles {
		logger, err := clog.LoggerFromConfigAsFile(file)
		if err != nil {
			println(err.Error())
			continue
		}
		clog.ReplaceLogger(logger)
		break
	}
}

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
	initLog()
	err := initDB()
	if err != nil {
		clog.Criticalf("Init DB failed: %s", err)
		panic("Shutdown")
	}
	// 设置拆包器
	pkg.SetUnpacker(pkg.NewDefaultUnpacker(50*1024*1024, 25))
	// 从数据库导入track
	pkg.JobMgr.LoadTracks()
}

func main() {
	start()
	clog.Info("Start")
	beego.Run(":8088")
}
