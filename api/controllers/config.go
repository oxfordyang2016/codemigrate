package controllers

import (
	"./../../pkg"
	"./../../utils"
	"cydex"
	clog "github.com/cihub/seelog"
)

type ConfigController struct {
	BaseController
}

func (self *ConfigController) Get() {
	rsp := new(cydex.ConfigRsp)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	// 非admin用户无权限
	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	rsp.F2tpFileMod = cydex.F2TP_FILE_MOD_SLICE
	if !pkg.IsUsingFileSlice() {
		rsp.F2tpFileMod = cydex.F2TP_FILE_MOD_NOSLICE
	}
	default_unpacker, ok := pkg.GetUnpacker().(*pkg.DefaultUnpacker)
	if ok {
		rsp.MinSegSize, rsp.MaxSegNum = default_unpacker.GetConfig()
		rsp.MinSegSize = rsp.MinSegSize / (1024 * 1024)
	}
}

func (self *ConfigController) Put() {
	rsp := new(cydex.BaseRsp)
	req := new(cydex.ConfigInfo)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	// 非admin用户无权限
	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	// 获取请求
	if err := self.FetchJsonBody(req); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	var is_file_slice bool
	switch req.F2tpFileMod {
	case cydex.F2TP_FILE_MOD_SLICE:
		is_file_slice = true
	case cydex.F2TP_FILE_MOD_NOSLICE:
		is_file_slice = false
	default:
		rsp.Error = cydex.ErrInvalidParam
		return
	}
	if pkg.IsUsingFileSlice() != is_file_slice {
		clog.Infof("Set file slice from %t to %t", pkg.IsUsingFileSlice(), is_file_slice)
		pkg.SetUsingFileSlice(is_file_slice)
		utils.DefaultConfig().SaveFileSlice(is_file_slice)
	}

	default_unpacker, ok := pkg.GetUnpacker().(*pkg.DefaultUnpacker)
	if !ok {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	min_seg_size, max_seg_num := default_unpacker.GetConfig()
	new_min_seg_size := req.MinSegSize * 1024 * 1024
	if min_seg_size != new_min_seg_size || max_seg_num != req.MaxSegNum {
		clog.Infof("Set unpacker config from (%d, %d) to (%d, %d)", min_seg_size, max_seg_num, new_min_seg_size, req.MaxSegNum)
		if err := default_unpacker.Config(new_min_seg_size, req.MaxSegNum); err != nil {
			rsp.Error = cydex.ErrInvalidParam
			return
		}
		utils.DefaultConfig().SaveDefaultUnpackerArgs(new_min_seg_size, req.MaxSegNum)
	}
}
