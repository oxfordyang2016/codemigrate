package controllers

import (
	"./../../pkg"
	pkg_model "./../../pkg/models"
	"./../../transfer/task"
	"cydex"
	"cydex/transfer"
	clog "github.com/cihub/seelog"
	"strconv"
	"time"
)

const (
	DISPATCH_TIMEOUT = 10 * time.Second
)

type TransferController struct {
	BaseController
}

func canUpload(user_level int) (ret bool) {
	if user_level == cydex.USER_LEVEL_COMMON || user_level == cydex.USER_LEVEL_UPLOAD_ONLY {
		ret = true
	}
	return
}

func canDownload(user_level int) (ret bool) {
	if user_level == cydex.USER_LEVEL_COMMON || user_level == cydex.USER_LEVEL_DOWNLOAD_ONLY {
		ret = true
	}
	return
}

func (self *TransferController) Post() {
	req := new(cydex.TransferReq)
	rsp := new(cydex.TransferRsp)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	// 获取请求
	if err := self.FetchJsonBody(req); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}
	req.FinishedSize, _ = strconv.ParseUint(req.FinishedSizeStr, 10, 64)

	if req.Fid == "" {
		clog.Error("Fid shouldn't be empty")
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	switch req.From {
	case "receiver":
		if !canDownload(self.UserLevel) {
			clog.Errorf("user level:%d not allowed to download", self.UserLevel)
			rsp.Error = cydex.ErrNotAllowed
			break
		}
		self.processDownload(req, rsp)
	case "sender":
		if !canUpload(self.UserLevel) {
			clog.Errorf("user level:%d not allowed to upload", self.UserLevel)
			rsp.Error = cydex.ErrNotAllowed
			break
		}
		self.processUpload(req, rsp)
	default:
		rsp.Error = cydex.ErrInvalidParam
		return
	}
}

func buildTaskDownloadReq(uid, pid, fid string, sids []string) *task.DownloadReq {
	task_req := new(task.DownloadReq)
	task_req.DownloadTaskReq = &transfer.DownloadTaskReq{
		TaskId:  task.GenerateTaskId(),
		Uid:     uid,
		Pid:     pid,
		Fid:     fid,
		SidList: sids,
	}

	file_m, _ := pkg_model.GetFile(fid)
	if file_m != nil {
		task_req.DownloadTaskReq.FileStorage = file_m.Storage
	}

	return task_req
}

func (self *TransferController) processDownload(req *cydex.TransferReq, rsp *cydex.TransferRsp) {
	// clog.Info("process download ")
	uid := self.GetString(":uid")
	pid := pkg.GetUnpacker().GetPidFromFid(req.Fid)

	jobid := pkg.HashJob(uid, pid, cydex.DOWNLOAD)
	// NOTE: 空seg数组表示客户端已有该Fid代表的文件
	if len(req.SegIds) == 0 {
		job := pkg.JobMgr.GetJob(jobid)
		jd := pkg.JobMgr.GetJobDetail(jobid, req.Fid)
		if jd != nil {
			jd.GetFile()
			if jd.File != nil {
				jd.FinishedSize = jd.File.Size
				jd.NumFinishedSegs = jd.File.NumSegs
			}
			jd.StartTime = time.Now()
			jd.FinishTime = jd.StartTime
			jd.State = cydex.TRANSFER_STATE_DONE
			jd.Save()
		}
		if job != nil {
			if job.IsCached {
				job.NumUnfinishedDetails--
			}
		}
		pkg.JobMgr.ProcessJob(jobid)
		return
	}
	// update jd process
	pkg.UpdateJobDetailProcess(jobid, req.Fid, req.FinishedSize, req.NumFinishedSegs)

	// jzh: 获取包裹所有者的uid
	var owner_uid string
	jobs, _ := pkg_model.GetJobsByPid(pid, cydex.UPLOAD, nil)
	if len(jobs) > 0 {
		owner_uid = jobs[0].Uid
	} else {
		clog.Errorf("pkg %s upload job not found", pid)
		rsp.Error = cydex.ErrInnerServer
		return
	}
	// get storages
	task_req := buildTaskDownloadReq(owner_uid, pid, req.Fid, req.SegIds)
	task_req.JobId = jobid
	trans_rsp, node, err := task.TaskMgr.DispatchDownload(task_req, DISPATCH_TIMEOUT)
	if err != nil || node == nil || trans_rsp == nil {
		clog.Error(err)
		rsp.Error = cydex.ErrInnerServer
		return
	}
	rsp.Error = trans_rsp.Rsp.Code
	if rsp.Error != cydex.OK {
		clog.Errorf("dispatch download task failed, code:%d", rsp.Error)
		return
	}

	task_rsp := trans_rsp.Rsp.DownloadTask

	rsp.Host = getHost(node.Host)
	rsp.Port = task_rsp.Port
	rsp.RecomendBitrate = task_rsp.RecomendBitrate
	for _, sid := range task_rsp.SidList {
		seg_m, _ := pkg_model.GetSeg(sid)
		if seg_m != nil {
			seg := new(cydex.Seg)
			seg.Sid = seg_m.Sid
			seg.SetSize(seg_m.Size)
			seg.Status = seg_m.State
			rsp.Segs = append(rsp.Segs, seg)
		}
		if rsp.Segs == nil {
			rsp.Segs = make([]*cydex.Seg, 0)
		}
	}

	clog.Infof("dispatch download ok, host:%s, port:%d, bps:%d", node.Host, task_rsp.Port, task_rsp.RecomendBitrate)
}

func (self *TransferController) processUpload(req *cydex.TransferReq, rsp *cydex.TransferRsp) {
	if len(req.SegIds) == 0 {
		clog.Error("Array of Seg id shouldn't be empty!")
		rsp.Error = cydex.ErrInvalidParam
		return
	}
	uid := self.GetString(":uid")
	file_m, err := pkg_model.GetFile(req.Fid)
	if err != nil || file_m == nil {
		clog.Error(err)
		rsp.Error = cydex.ErrInnerServer
		return
	}
	pid := file_m.Pid
	pkg_m, err := pkg_model.GetPkg(pid, false)
	if err != nil || pkg_m == nil {
		clog.Error(err)
		rsp.Error = cydex.ErrInnerServer
		return
	}

	task_req := new(task.UploadReq)
	task_req.JobId = pkg.HashJob(uid, pid, cydex.UPLOAD)
	task_req.FileSize = file_m.Size
	task_req.PkgSize = pkg_m.Size
	task_req.UploadTaskReq = &transfer.UploadTaskReq{
		TaskId:  task.GenerateTaskId(),
		Uid:     uid,
		Pid:     pid,
		Fid:     req.Fid,
		SidList: req.SegIds,
	}
	for _, sid := range req.SegIds {
		seg, _ := pkg_model.GetSeg(sid)
		if seg != nil {
			task_req.UploadTaskReq.Size += seg.Size
		}
	}

	clog.Tracef("pid:%s, pkg_size:%d, file_size:%d,segs size:%d", task_req.Pid, task_req.PkgSize, task_req.FileSize, task_req.UploadTaskReq.Size)

	trans_rsp, node, err := task.TaskMgr.DispatchUpload(task_req, DISPATCH_TIMEOUT)
	if err != nil || node == nil || trans_rsp == nil {
		clog.Error(err)
		rsp.Error = cydex.ErrInnerServer
		return
	}
	if trans_rsp.Rsp.Code != cydex.OK {
		clog.Errorf("dispatch upload failed, code:%d, reason:%s ", trans_rsp.Rsp.Code, trans_rsp.Rsp.Reason)
		rsp.Error = trans_rsp.Rsp.Code
		return
	}

	task_rsp := trans_rsp.Rsp.UploadTask
	// update storage
	file_m.SetStorage(task_rsp.FileStorage)

	rsp.Host = getHost(node.Host)
	rsp.Port = task_rsp.Port
	rsp.RecomendBitrate = task_rsp.RecomendBitrate
	for _, sid := range task_rsp.SidList {
		seg_m, _ := pkg_model.GetSeg(sid)
		if seg_m != nil {
			seg := new(cydex.Seg)
			seg.Sid = seg_m.Sid
			seg.SetSize(seg_m.Size)
			seg.Status = seg_m.State
			rsp.Segs = append(rsp.Segs, seg)
		}
		if rsp.Segs == nil {
			rsp.Segs = make([]*cydex.Seg, 0)
		}
	}

	clog.Infof("dispatch upload ok, host:%s, port:%d, bps:%d", node.Host, task_rsp.Port, task_rsp.RecomendBitrate)
}

// 如果是"127.0.0.1", 则返回空,兼容单机版
func getHost(host string) string {
	if host == "127.0.0.1" {
		return ""
	}
	return host
}
