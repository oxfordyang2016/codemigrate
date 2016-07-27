package controllers

import (
	"./../../pkg"
	pkg_model "./../../pkg/models"
	trans "./../../transfer"
	"./../../transfer/task"
	"cydex"
	"cydex/transfer"
	"encoding/json"
	"github.com/astaxie/beego"
	"time"
)

const (
	DISPATCH_TIMEOUT = 10 * time.Second
)

type TransferController struct {
	beego.Controller
}

func (self *TransferController) Post() {
	req := new(transfer.TransferReq)
	rsp := new(transfer.TransferRsp)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	// 获取请求
	if err := json.Unmarshal(self.Ctx.Input.RequestBody, req); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	if len(req.SegIds) == 0 || req.Fid == "" {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	switch req.From {
	case "receiver":
		self.processDownload(req, rsp)
	case "sender":
	default:
		rsp.Error = cydex.ErrInvalidParam
		return
	}
}

func (self *TransferController) processDownload(req *cydex.TransferReq, rsp *cydex.TransferRsp) {
	task_req := new(task.DownloadReq)
	task_req.Pid = pkg.GetUnpacker().GetPidFromFid(req.Fid)
	task_req.DownloadReq = &transfer.DownloadTaskReq{
		TaskId:  task.GenerateTaskId(),
		Uid:     uid,
		Fid:     req.Fid,
		SidList: req.SegIds,
	}
	// get storages
	for _, sid := range req.SegIds {
		seg, _ := pkg_model.GetSeg(sid)
		storage := ""
		if seg != nil {
			storage = seg.Storage
		}
		task_req.DownloadReq.SidStorage = append(task_req.DownloadReq.SidStorage, storage)
	}

	var err error
	var task_rsp *transfer.DownloadTaskRsp
	var node *trans.Node

	if task_rsp, node, err = task.DispatchDownload(task_req, DISPATCH_TIMEOUT); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	rsp.Host = node.Host
	rsp.Port = task_rsp.Port
	rsp.Segs = task_rsp.SidList
	rsp.RecomendBitrate = task_rsp.RecomendBitrate
}

func (self *TransferController) processUpload(req *cydex.TransferReq, rsp *cydex.TransferRsp) {
	task_req := new(task.UploadReq)
	task_req.Pid = pkg.GetUnpacker().GetPidFromFid(req.Fid)
	task_req.UploadReq = &transfer.UploadTaskReq{
		TaskId:  task.GenerateTaskId(),
		Uid:     uid,
		Fid:     req.Fid,
		SidList: req.SegIds,
	}
	for _, sid := range req.SegIds {
		seg, _ := pkg_model.GetSeg(sid)
		if seg != nil {
			task_req.UploadReq.Size += seg.Size
		}
	}

	task_rsp, node, err := task.DispatchUpload(task_req, DISPATCH_TIMEOUT)
	if err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}

	rsp.Host = node.Host
	rsp.Port = task_rsp.Port
	rsp.Segs = task_rsp.SidList
	rsp.RecomendBitrate = task_rsp.RecomendBitrate
}
