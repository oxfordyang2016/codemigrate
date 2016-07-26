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
	uid := self.GetString(":uid")
	query := self.GetString("query")
	filter := self.GetString("filter")

	switch query {
	case "all":
		if filter == "change" {
			self.getActive()
		}
	case "sender":
		self.getUploads()
	case "receiver":
		self.getDownloads()
	default:
		self.getAll()
	}
}

//
func (self *PkgsController) getActive() {

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
