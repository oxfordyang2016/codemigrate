package main

import (
	_ "./api/"
	"./db"
	"./pkg"
	pkg_model "./pkg/models"
	// trans_model "./transfer/models"
	"github.com/astaxie/beego"
)

func start() {
	// 创建数据库
	db.CreateDefaultDBEngine("sqlite3", "/tmp/cydex.sqlite3", false)
	db.DB().SyncModels(pkg_model.Models)
	// 设置拆包器
	pkg.SetUnpacker(pkg.NewDefaultUnpacker(50*1024*1024, 25))
}

func main() {
	start()
	beego.Run()
}
