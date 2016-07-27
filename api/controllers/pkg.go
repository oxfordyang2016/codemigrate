package controllers

import (
	"./../../pkg"
	pkg_model "./../../pkg/models"
	"cydex"
	"encoding/json"
	"github.com/astaxie/beego"
	"time"
)

// 包控制器
type PkgsController struct {
	beego.Controller
}

func (self *PkgsController) Get() {
	query := self.GetString("query")
	filter := self.GetString("filter")

	switch query {
	case "all":
		if filter == "change" {
			self.getActive()
		}
	case "sender":
		self.getJobs(cydex.UPLOAD)
	case "receiver":
		self.getJobs(cydex.DOWNLOAD)
	}
}

func (self *PkgsController) getJobs(typ int) {
	uid := self.GetString(":uid")
	page_size := self.GetInt("page_size")
	page_num := self.GetInt("page_num")
	// 判断是否是admin
	// 目前是普通用户处理
	rsp := new(cydex.QueryPkgRsp)
	rsp.Error = cydex.OK

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	// 按照uid和type得到和用户相关的jobs
	jobs := pkg_model.GetJobsByUid(uid, typ)
	for _, job := range jobs {
		pkg := new(cydex.Pkg)

		// 获取pkg信息
		pkg_m, err := pkg_model.GetPkg(job.Pid, false)
		if err != nil {
			// error
			rsp.Error = cydex.ErrInnerServer
			return
		}
		pkg.Pid = pkg_m.Pid
		pkg.Title = pkg_m.Title
		pkg.Notes = pkg_m.Notes
		pkg.NumFiles = pkg_m.NumFiles
		pkg.Date = pkg_m.CreateAt
		pkg.EncryptionType = pkg_m.EncryptionType

		if err = pkg_m.GetFiles(true); err != nil {
			// error
			rsp.Error = cydex.ErrInnerServer
			return
		}

		// 根据pid获取下载该pkg的信息
		download_jobs, err := pkg_model.GetJobsByPid(job.Pid, cydex.DOWNLOAD)
		if err != nil {
			// error
			rsp.Error = cydex.ErrInnerServer
			return
		}
		// 根据pid获取上传该pkg的信息
		upload_jobs, err := pkg_model.GetJobsByPid(job.Pid, cydex.UPLOAD)
		if err != nil {
			// error
			rsp.Error = cydex.ErrInnerServer
		}

		for _, file_m := range pkg_m.Files {
			file := new(cydex.File)
			file.Fid = file_m.Fid
			file.Filename = file_m.Filename
			file.Size = file.Size
			file.Path = file_m.Path
			file.Type = file_m.Type
			file.Chara = file_m.EigenValue
			file.PathAbs = file_m.PathAbs

			// 获取该文件的关联信息
			for _, d_jobs := range download_jobs {
				jd, err := pkg_model.GetJobDetail(d_jobs.JobId, file.Fid)
				if err != nil {
					// error
					rsp.Error = cydex.ErrInnerServer
					return
				}
				state := new(cydex.TransferState)
				state.Uid = d_jobs.Uid
				state.State = jd.State
				state.Percent = jd.FinishedSize * 100 / file.Size
				if !jd.StartTime.IsZero() {
					t, err := jd.StartTime.MarshalText()
					state.StartTime = string(t)
				}
				if !jd.FinishTime.IsZero() {
					t, err := jd.FinishTime.MarshalText()
					state.FinishTime = string(t)
				}
				file.DownloadState = append(file.DownloadState, state)
			}

			for _, u_jobs := range upload_jobs {
				jd, err := pkg_model.GetJobDetail(u_jobs.JobId, file.Fid)
				if err != nil {
					// error
					rsp.Error = cydex.ErrInnerServer
					return
				}
				state := new(cydex.TransferState)
				state.Uid = u_jobs.Uid
				state.State = jd.State
				state.Percent = jd.FinishedSize * 100 / file.Size
				if !jd.StartTime.IsZero() {
					t, err := jd.StartTime.MarshalText()
					state.StartTime = string(t)
				}
				if !jd.FinishTime.IsZero() {
					t, err := jd.FinishTime.MarshalText()
					state.FinishTime = string(t)
				}
				file.UploadState = append(file.UploadState, state)
			}

			for _, seg_m := range file_m.Segs {
				seg := new(cydex.Seg)
				seg.Sid = seg_m.Sid
				seg.SetSize(seg_m.Size)
				seg.Status = seg_m.Status
				file.Segs = append(file.Segs, seg)
			}
			pkg.Files = append(pkg.Files, file)
		}

		rsp.Pkgs = append(rsp.Pkgs, pkg)
	}
}

//
func (self *PkgsController) getActive() {
	uid := self.GetString(":uid")
	rsp := new(cydex.QueryActivePkgRsp)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	// upload
	pids, err := pkg.JobMgr.GetUserTrack(uid, cydex.UPLOAD)
	if err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	for _, pid := range pids {
		pkg_u := new(cydex.PkgUpload)
		pkg_u.Pid = pid
		job_m := pkg.JobMgr.GetJob(pkg.HashJob(uid, pid, cydex.UPLOAD))
		if job_m == nil || job_m.Pkg == nil {
			// can't be true
		}
		for _, file_m := range job_m.Pkg {
			file := new(cydex.FileReceiver)
			file.Fid = file_m.Fid
			file.Filename = file_m.Filename
		}
	}
}

func (self *PkgsController) Post() {
	rsp := new(cydex.CreatePkgRsp)
	req := new(cydex.CreatePkgReq)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	// 获取拆包器
	unpacker := pkg.GetUnpacker()
	if unpacker == nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	// 获取请求
	if err := json.Unmarshal(self.Ctx.Input.RequestBody, req); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	self.createPkg(req, rsp)
}

// 创建包裹
func (self *PkgsController) createPkg(req *cydex.CreatePkgReq, rsp *cydex.CreatePkgRsp) {
	if !req.Verify() {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	var (
		err           error
		uid, fid, pid string
		file_o        *cydex.File
	)
	session := pkg_model.DB().NewSession()
	session.Begin()

	defer func() {
		if err != nil {
			session.Rollback()
		}
		session.Close()
	}()

	pkg_o := new(cydex.Pkg)
	pid = unpacker.GeneratePid(uid, req.Title, req.Notes)
	size := uint64(0)
	for _, f := range req.Files {
		size += f.Size
	}

	// 创建pkg数据库记录
	pkg_m := &pkg_model.Pkg{
		Pid:            pid,
		Title:          req.Title,
		Notes:          req.Notes,
		NumFiles:       uint64(len(req.Files)),
		Size:           size,
		EncryptionType: req.EncryptionType,
	}
	if _, err = session.Insert(pkg_m); err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}

	pkg_o.Pid = pid
	pkg_o.Title = req.Title
	pkg_o.Notes = req.Notes
	pkg_o.Date = time.Now()
	pkg_o.NumFiles = len(req.Files)

	for i, f := range req.Files {
		if fid, err = unpacker.GenerateFid(pid, i, f); err != nil {
			rsp.Error = cydex.ErrInnerServer
			return
		}
		file_o = new(cydex.File)
		file_o.Fid = fid
		file_o.SimpleFile = *f
		file_o.Segs, err = unpacker.GenerateSegs(fid, f)
		if err != nil {
			rsp.Error = cydex.ErrInnerServer
			return
		}
		file_m := &pkg_model.File{
			Fid:      fid,
			Pid:      pid,
			Filename: f.Filename,
			Path:     f.Path,
			Size:     f.Size,
			NumSegs:  len(file_o.Segs),
		}
		if _, err = session.Insert(file_m); err != nil {
			rsp.Error = cydex.ErrInnerServer
			return
		}
		for _, s := range file_o.Segs {
			seg_m := &pkg_model.Seg{
				Sid:  s.Sid,
				Fid:  fid,
				Size: s.InnerSize,
			}
			if _, err = session.Insert(seg_m); err != nil {
				rsp.Error = cydex.ErrInnerServer
				return
			}
		}
		pkg_o.Files = append(pkg_o.Files, file_o)
	}
	session.Commit()
	rsp.Pkg = pkg_o

	// Dispatch jobs
	pkg.JobMgr.CreateJob(uid, pid, cydex.UPLOAD)
	if req.Receivers != nil {
		for _, r := range req.Receivers {
			pkg.JobMgr.CreateJob(r.Uid, pid, cydex.DOWNLOAD)
		}
	}
}

// 单包控制器
type PkgController struct {
	beego.Controller
}

func (self *PkgController) Get() {
}

// 单文件控制器
type FileController struct {
	beego.Controller
}

func (self *FileController) Get() {
}
