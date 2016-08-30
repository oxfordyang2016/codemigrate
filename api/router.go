package api

import (
	c "./controllers"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	clog "github.com/cihub/seelog"
)

func setupRoute() {
	// TODO 正则限制长度
	beego.Router("/:uid/pkg", &c.PkgsController{})
	beego.Router("/:uid/pkg/:pid", &c.PkgController{})
	beego.Router("/:uid/pkg/:pid/:fid", &c.FileController{})
	beego.Router("/:uid/transfer", &c.TransferController{})
	beego.Router("/:uid/log", &c.LogController{})
}

func setupFilter() {
	beego.InsertFilter("/*", beego.BeforeRouter, func(ctx *context.Context) {
		clog.Tracef("[http] %s %s [%s]", ctx.Request.Method, ctx.Request.URL, ctx.Request.RemoteAddr)
	})
}

func init() {
	setupFilter()
	setupRoute()
}
