package api

import (
	c "./controllers"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	clog "github.com/cihub/seelog"
)

func setupRoute() {
	beego.Router("/:uid/pkg", &c.PkgsController{})
	beego.Router("/:uid/pkg/:pid", &c.PkgController{})
	beego.Router("/:uid/pkg/:pid/:fid", &c.FileController{})
	beego.Router("/:uid/transfer", &c.TransferController{})
	beego.Router("/:uid/log", &c.LogController{})

	beego.Router("/api/v1/nodes", &c.NodesController{})
	beego.Router("/api/v1/nodes/:id", &c.NodeController{})
	beego.Router("/api/v1/nodes/:id/status", &c.NodeStatusController{})
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
