package api

import (
	c "./controllers"
	"github.com/astaxie/beego"
)

func setup() {
	// TODO 正则限制长度
	beego.Router("/:uid/pkg", &c.PkgsController{})
	beego.Router("/:uid/pkg/:pid", &c.PkgController{})
	beego.Router("/:uid/pkg/:pid/:fid", &c.FileController{})
	beego.Router("/:uid/transfer", &c.TransferController{})
}

func init() {
	setup()
}
