package controllers

import (
	trans "./../../transfer"
	"./../../transfer/models"
	"cydex"
	"cydex/transfer"
	clog "github.com/cihub/seelog"
)

type NodesController struct {
	BaseController
}

func (self *NodesController) Get() {
	page := new(cydex.Pagination)
	page.PageSize, _ = self.GetInt("page_size")
	page.PageNum, _ = self.GetInt("page_num")
	if !page.Verify() {
		page = nil
	}

	rsp := new(cydex.QueryNodeListRsp)
	rsp.Error = cydex.OK

	defer func() {
		if rsp.Nodes == nil {
			rsp.Nodes = make([]*cydex.Node, 0)
		}
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	nodes_m, err := models.GetNodes(page)
	if err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}

	total, err := models.CountNodes()
	if err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	rsp.TotalNum = int(total)
	for _, n := range nodes_m {
		node := getSignalNode(n)
		if node != nil {
			rsp.Nodes = append(rsp.Nodes, node)
		}
	}
}

func getSignalNode(node_m *models.Node) *cydex.Node {
	node := new(cydex.Node)

	node.Id = node_m.Nid
	node.Name = node_m.Name
	node.RxBandwidth = node_m.RxBandwidth
	node.TxBandwidth = node_m.TxBandwidth
	node.PublicAddr = node_m.PublicAddr
	node.ZoneId = node_m.ZoneId
	node_m.GetZone()
	if node_m.Zone != nil {
		node.ZoneName = node_m.Zone.Name
	}
	node.MachineCode = node_m.MachineCode
	node.RegisterAt = MarshalUTCTime(node_m.RegisterTime)
	node.Online = false

	node_live := trans.NodeMgr.GetByNid(node_m.Nid)
	if node_live != nil {
		node.Online = true
		node.OnlineAt = MarshalUTCTime(node_live.OnlineAt())
		node.RemoteAddr = node_live.Host
		node.Version = node_live.Info.Version
		node.F2tpVersion = node_live.Info.F2tpVersion
		node.UploadTaskCnt = node_live.Info.UploadTaskCnt
		node.DownloadTaskCnt = node_live.Info.DownloadTaskCnt
	}
	return node
}

type NodeController struct {
	BaseController
}

func (self *NodeController) Get() {
	rsp := new(cydex.QueryNodeRsp)
	rsp.Error = cydex.OK

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	node_id := self.GetString(":id")
	node_m, _ := models.GetNode(node_id)
	if node_m == nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	rsp.Node = getSignalNode(node_m)
}

func (self *NodeController) Put() {
	self.Patch()
}

func (self *NodeController) Patch() {
	req := new(cydex.NodeModify)
	rsp := new(cydex.BaseRsp)
	rsp.Error = cydex.OK

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	nid := self.GetString(":id")
	node_m, _ := models.GetNode(nid)
	if node_m == nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	// 获取请求
	if err := self.FetchJsonBody(req); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	var err error
	err_cnt := 0
	if req.Name != nil {
		if err = node_m.SetName(*req.Name); err != nil {
			err_cnt++
		}
	}
	if req.PublicAddr != nil {
		if err = node_m.SetPublicAddr(*req.PublicAddr); err != nil {
			err_cnt++
		}
	}

	//设置带宽
	var rx, tx uint64
	if req.RxBandwidth != nil {
		rx = *req.RxBandwidth
	}
	if req.TxBandwidth != nil {
		tx = *req.TxBandwidth
	}
	if rx > 0 || tx > 0 {
		n_live := trans.NodeMgr.GetByNid(nid)
		if n_live == nil {
			clog.Warnf("Config rx/tx bandwidth, node(%s) is OFFLINE", nid)
		} else {
			msg := transfer.NewReqMessage("", "config", "", 0)
			msg.Req.Config = &transfer.ConfigReq{
				RxBandwidth: req.RxBandwidth,
				TxBandwidth: req.TxBandwidth,
			}
			rsp, err := n_live.SendRequestSync(msg, DISPATCH_TIMEOUT)
			// node设置ok了,再设进数据库,保持一致
			clog.Tracef("%+v", rsp)
			if err == nil && rsp != nil && rsp.Rsp.Code == cydex.OK {
				if rx > 0 {
					if err = node_m.SetRxBandwidth(rx); err != nil {
						err_cnt++
					}
				}
				if tx > 0 {
					if err = node_m.SetTxBandwidth(tx); err != nil {
						err_cnt++
					}
				}
			} else {
				err_cnt++
			}
		}
	}

	if err_cnt > 0 {
		rsp.Error = cydex.ErrInvalidParam
	}
}

type NodeStatusController struct {
	BaseController
}

func (self *NodeStatusController) Get() {
	rsp := new(cydex.QueryNodeStatusRsp)
	rsp.Error = cydex.OK

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	node_id := self.GetString(":id")
	node_live := trans.NodeMgr.GetByNid(node_id)
	if node_live != nil {
		rsp.Status = &cydex.NodeStatus{
			Uptime:          uint64(node_live.OnlineDuration().Seconds()),
			UploadTaskCnt:   node_live.Info.UploadTaskCnt,
			DownloadTaskCnt: node_live.Info.DownloadTaskCnt,
			CpuUsage:        node_live.Info.CpuUsage,
			TotalMem:        node_live.Info.TotalMem,
			FreeMem:         node_live.Info.FreeMem,
			TotalStorage:    node_live.Info.TotalStorage,
			FreeStorage:     node_live.Info.FreeStorage,
			RxBandwidth:     node_live.Info.RxBandwidth,
			TxBandwidth:     node_live.Info.TxBandwidth,
		}
	}
}
