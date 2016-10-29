package controllers

import (
	"./../../pkg"
	pkg_model "./../../pkg/models"
	trans "./../../transfer"
	trans_model "./../../transfer/models"
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

// api 3.9
// TODO 要记录上传的片段size和状态
func (self *TransferController) Get() {
	rsp := new(cydex.QueryTransferSegs)
	uid := self.GetString(":uid")
	// query := self.GetString("query")
	fid := self.GetString("fid")

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	file_m, _ := pkg_model.GetFile(fid)
	if file_m == nil {
		rsp.Error = cydex.ErrFileNotExisted
		return
	}
	jobid := pkg.HashJob(uid, file_m.Pid, cydex.UPLOAD)
	job_m, _ := pkg_model.GetJob(jobid, false)
	if job_m == nil {
		rsp.Error = cydex.ErrFileNotExisted
		return
	}
	if err := file_m.GetSegs(); err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	for _, seg_m := range file_m.Segs {
		seg := new(cydex.Seg)
		seg.Status = seg_m.State
		seg.Sid = seg_m.Sid
		if seg_m.State != cydex.TRANSFER_STATE_DONE {
			seg.SetSize(0)
		} else {
			seg.SetSize(seg_m.Size)
		}
		rsp.Segs = append(rsp.Segs, seg)
	}
	if rsp.Segs == nil {
		rsp.Segs = make([]*cydex.Seg, 0)
	}
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
	if req.MaxBitrateStr != "" {
		req.MaxBitrate, _ = strconv.ParseUint(req.MaxBitrateStr, 10, 64)
	}

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

func buildTaskDownloadReq(uid, pid, fid string, sids []string, max_bitrate uint64) *task.DownloadReq {
	file_m, _ := pkg_model.GetFile(fid)
	detail := getFileDetail(file_m)
	if cydex.IsFileNoSlice(file_m.Flag) && detail == nil {
		clog.Errorf("%s flag is %d, detail shouldn't be nil", file_m, file_m.Flag)
		return nil
	}

	task_req := new(task.DownloadReq)
	task_req.DownloadTaskReq = &transfer.DownloadTaskReq{
		TaskId:     task.GenerateTaskId(),
		Uid:        uid,
		Pid:        pid,
		Fid:        fid,
		SidList:    sids,
		FileDetail: detail,
		MaxBitrate: max_bitrate,
	}
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
	task_req := buildTaskDownloadReq(owner_uid, pid, req.Fid, req.SegIds, req.MaxBitrate)
	if task_req == nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
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

	rsp.Host = getHost(node)
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
	jobid := pkg.HashJob(uid, pid, cydex.UPLOAD)
	job_m := pkg.JobMgr.GetJob(jobid)
	if job_m == nil {
		clog.Error(err)
		rsp.Error = cydex.ErrInnerServer
		return
	}
	transferd_size, _ := job_m.GetTransferedSize()
	detail := getFileDetail(file_m)
	if cydex.IsFileNoSlice(file_m.Flag) && detail == nil {
		clog.Errorf("%s flag is %d, detail shouldn't be nil", file_m, file_m.Flag)
		rsp.Error = cydex.ErrInnerServer
	}

	task_req := new(task.UploadReq)
	task_req.JobId = jobid
	task_req.FileSize = file_m.Size
	task_req.LeftPkgSize = pkg_m.Size - transferd_size
	task_req.UploadTaskReq = &transfer.UploadTaskReq{
		TaskId:      task.GenerateTaskId(),
		Uid:         uid,
		Pid:         pid,
		Fid:         req.Fid,
		SidList:     req.SegIds,
		FileDetail:  detail,
		FileStorage: file_m.Storage, //issue-31,48
	}
	for _, sid := range req.SegIds {
		seg, _ := pkg_model.GetSeg(sid)
		if seg != nil {
			task_req.UploadTaskReq.Size += seg.Size
		}
	}

	clog.Tracef("upload req: %+v", task_req)

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
	if file_m.Storage == "" {
		file_m.SetStorage(task_rsp.FileStorage)
	} else {
		if file_m.Storage != task_rsp.FileStorage {
			clog.Warnf("File's storage url:'%s' is different from the node provides '%s'", file_m.Storage, task_rsp.FileStorage)
		}
	}

	rsp.Host = getHost(node)
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

// 如果有预设的public_addr, 则取之
// 无public_addr则取node的远程地址,如果为"127.0.0.1"则返回空,兼容单机版协议
func getHost(node *trans.Node) string {
	var public_addr string
	node_m, _ := trans_model.GetNode(node.Nid)
	if node_m != nil {
		public_addr = node_m.PublicAddr
	}
	if public_addr != "" {
		return public_addr
	}
	if node.Host == "127.0.0.1" {
		return ""
	}
	return node.Host
}

func getFileDetail(file *pkg_model.File) *transfer.FileDetail {
	if file == nil {
		return nil
	}
	if cydex.IsFileSlice(file.Flag) {
		return nil
	}
	p, _ := pkg_model.GetPkg(file.Pid, false)
	if p == nil {
		return nil
	}
	segs, _ := pkg_model.GetSegs(file.Fid)
	if segs == nil {
		return nil
	}
	detail := &transfer.FileDetail{
		FileName:       file.Name,
		FileFlag:       file.Flag,
		FileMode:       uint32(file.Mode),
		FileSize:       file.Size,
		EncryptionType: p.EncryptionType,
		SegCnt:         file.NumSegs,
	}
	if p.MetaData != nil {
		detail.MtuSize = p.MetaData.MtuSize
	}
	seg := segs[0]
	detail.SegPerSize = seg.Size
	s := string(seg.Sid[len(seg.Sid)-1])
	detail.SegStartNo, _ = strconv.Atoi(s)

	clog.Debugf("%+v", detail)

	return detail
}
