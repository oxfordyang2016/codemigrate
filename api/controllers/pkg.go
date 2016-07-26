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
	// uid := self.Ctx.Input.Param(":uid")
}

// 创建包裹
// FIXME 需要事务处理?
func (self *PkgsController) Post() {
	rsp := new(cydex.CreatePkgRsp)
	req := new(cydex.CreatePkgReq)
	pkg_o := new(cydex.Pkg)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	var (
		err           error
		uid, fid, pid string
		// pkg_m         *pkg_model.Pkg
		// file_m        *pkg_model.File
		// seg_m         *pkg_model.Seg
		file_o *cydex.File
	)

	uid = self.Ctx.Input.Param(":uid")
	// 获取拆包器
	unpacker := pkg.GetUnpacker()
	if unpacker == nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	// 获取请求
	if err = json.Unmarshal(self.Ctx.Input.RequestBody, req); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}
	if !req.Verify() {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	pid = unpacker.GeneratePid(uid, req.Title, req.Notes)
	size := uint64(0)
	for _, f := range req.Files {
		size += f.Size
	}
	if _, err := pkg_model.CreatePkg(pid, req.Title, req.Notes, uint64(len(req.Files)), size, req.EncryptionType); err != nil {
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
		if _, err = pkg_model.CreateFile(fid, f.Filename, f.Path, f.Size, len(file_o.Segs)); err != nil {
			rsp.Error = cydex.ErrInnerServer
			return
		}
		for _, s := range file_o.Segs {
			if _, err = pkg_model.CreateSeg(s.Sid, fid, s.InnerSize); err != nil {
				rsp.Error = cydex.ErrInnerServer
				return
			}
		}
		pkg_o.Files = append(pkg_o.Files, file_o)
	}
	rsp.Pkg = pkg_o

	// Dispatch jobs
	pkg.JobMgr.NewJob(uid, pid, cydex.UPLOAD)
	if req.Receivers != nil {
		for _, r := range req.Receivers {
			pkg.JobMgr.NewJob(r.Uid, pid, cydex.DOWNLOAD)
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
