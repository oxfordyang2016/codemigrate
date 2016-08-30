package controllers

import (
	trans "./../../transfer"
	"./../../utils"
	"cydex"
	clog "github.com/cihub/seelog"
)

type LogController struct {
	BaseController
}

func (self *LogController) Get() {
	query := self.GetString("query")
	switch query {
	case "disk":
		self.getDiskInfo()
	default:
		return
	}
}

func (self *LogController) getDiskInfo() {
	rsp := new(cydex.GetStorageRsp)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	var total, free uint64

	// 管理员
	if self.UserLevel == cydex.USER_LEVEL_ADMIN {
		trans.NodeMgr.WalkNodes(func(n *trans.Node) {
			total += n.Info.TotalStorage
			free += n.Info.FreeStorage
		})
	} else {
		trans.NodeMgr.WalkNodes(func(n *trans.Node) {
			total += n.Info.TotalStorage
			free += n.Info.FreeStorage
		})
	}

	rsp.SpaceStr = utils.GetHumanSize(free)
	rsp.TotalStr = utils.GetHumanSize(total)
	rsp.UsedStr = utils.GetHumanSize(total - free)

	clog.Infof("storage size: %s, free:%s", rsp.TotalStr, rsp.SpaceStr)
	return
}
