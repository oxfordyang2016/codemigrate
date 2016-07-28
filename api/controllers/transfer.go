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
	"io/ioutil"
	"time"
)

const (
	DISPATCH_TIMEOUT = 10 * time.Second
)

type TransferController struct {
	beego.Controller
}

func (self *TransferController) Post() {
	req := new(cydex.TransferReq)
	rsp := new(cydex.TransferRsp)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	r := self.Ctx.Request
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}
	// 获取请求
	if err := json.Unmarshal(body, req); err != nil {
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
	uid := self.GetString(":uid")
	task_req := new(task.DownloadReq)
	task_req.Pid = pkg.GetUnpacker().GetPidFromFid(req.Fid)
	task_req.DownloadTaskReq = &transfer.DownloadTaskReq{
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
		task_req.DownloadTaskReq.SidStorage = append(task_req.DownloadTaskReq.SidStorage, storage)
	}

	var err error
	var task_rsp *transfer.DownloadTaskRsp
	var node *trans.Node

	if task_rsp, node, err = task.TaskMgr.DispatchDownload(task_req, DISPATCH_TIMEOUT); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	rsp.Host = node.Host
	rsp.Port = task_rsp.Port
	rsp.Segs = task_rsp.SidList
	rsp.RecomendBitrate = task_rsp.RecomendBitrate
}

func (self *TransferController) processUpload(req *cydex.TransferReq, rsp *cydex.TransferRsp) {
	uid := self.GetString(":uid")
	task_req := new(task.UploadReq)
	task_req.Pid = pkg.GetUnpacker().GetPidFromFid(req.Fid)
	task_req.UploadTaskReq = &transfer.UploadTaskReq{
		TaskId:  task.GenerateTaskId(),
		Uid:     uid,
		Fid:     req.Fid,
		SidList: req.SegIds,
	}
	for _, sid := range req.SegIds {
		seg, _ := pkg_model.GetSeg(sid)
		if seg != nil {
			task_req.UploadTaskReq.Size += seg.Size
		}
	}

	task_rsp, node, err := task.TaskMgr.DispatchUpload(task_req, DISPATCH_TIMEOUT)
	if err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}

	rsp.Host = node.Host
	rsp.Port = task_rsp.Port
	rsp.Segs = task_rsp.SidList
	rsp.RecomendBitrate = task_rsp.RecomendBitrate
}
