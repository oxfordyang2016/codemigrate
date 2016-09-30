package controllers

import (
	"./../../statistics"
	"cydex"
	// clog "github.com/cihub/seelog"
)

type StatTransferController struct {
	BaseController
}

func (self *StatTransferController) Get() {
	rsp := new(cydex.QueryStatisticsTransferRsp)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	// 管理员
	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	node_id := self.GetString(":node_id")
	if node_id == "" {
		self.getOutline(rsp)
	} else {
		self.getSingle(node_id, rsp)
	}
}

func (self *StatTransferController) getOutline(rsp *cydex.QueryStatisticsTransferRsp) {
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
	rsp.RxStat = rx_stat
	rsp.TxStat = tx_stat
}

func (self *StatTransferController) getSingle(node_id string, rsp *cydex.QueryStatisticsTransferRsp) {
	rx_stat := statistics.TransferMgr.GetNodeStat(node_id, cydex.UPLOAD)
	if rx_stat == nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	tx_stat := statistics.TransferMgr.GetNodeStat(node_id, cydex.DOWNLOAD)
	if tx_stat == nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	rsp.RxStat = rx_stat
	rsp.TxStat = tx_stat
}

// TODO
type StatPkgController struct {
	BaseController
}

func (self *StatPkgController) Get() {
	rsp := new(cydex.QueryStatisticsPkgRsp)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	// 管理员
	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	stat_pkg := statistics.PkgMgr.Get()
	rsp.PkgStat = stat_pkg
}
