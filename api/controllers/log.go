package controllers

import (
	"./../../statistics"
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
	case "server_info":
		self.getServerInfo()
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

	total, free = trans.CalcStorage()
	// 管理员
	// if self.UserLevel == cydex.USER_LEVEL_ADMIN {
	// 	trans.NodeMgr.WalkNodes(func(n *trans.Node) {
	// 		total += n.Info.TotalStorage
	// 		free += n.Info.FreeStorage
	// 	})
	// } else {
	// 	trans.NodeMgr.WalkNodes(func(n *trans.Node) {
	// 		total += n.Info.TotalStorage
	// 		free += n.Info.FreeStorage
	// 	})
	// }

	rsp.SpaceStr = utils.GetHumanSize(free)
	rsp.TotalStr = utils.GetHumanSize(total)
	rsp.UsedStr = utils.GetHumanSize(total - free)

	clog.Infof("storage size: %s, free:%s", rsp.TotalStr, rsp.SpaceStr)
	return
}

func (self *LogController) getServerInfo() {
	rsp := new(cydex.GetServerInfoRsp)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	// 管理员
	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	rx_stat := statistics.TransferMgr.GetStat(cydex.UPLOAD)
	if rx_stat == nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	tx_stat := statistics.TransferMgr.GetStat(cydex.DOWNLOAD)
	if tx_stat == nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	rsp.DownloadTraffic = tx_stat.TotalBytes
	rsp.UploadTraffic = rx_stat.TotalBytes
	rsp.Bandwidth.MaxOut = tx_stat.MaxBitrate
	rsp.Bandwidth.MaxIn = rx_stat.MaxBitrate
	rsp.Bandwidth.OutBandwidth = statistics.TransferMgr.GetCurBitrate(cydex.DOWNLOAD)
	rsp.Bandwidth.InBandwidth = statistics.TransferMgr.GetCurBitrate(cydex.UPLOAD)

	clog.Tracef("get server info: %+v, %+v", rsp, rsp.Bandwidth)
	return
}
