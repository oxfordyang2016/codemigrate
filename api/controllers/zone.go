package controllers

import (
	"./../../transfer/models"
	"cydex"
	// "cydex/transfer"
	clog "github.com/cihub/seelog"
	"github.com/pborman/uuid"
)

type ZonesController struct {
	BaseController
}

func (self *ZonesController) Get() {
	page := new(cydex.Pagination)
	page.PageSize, _ = self.GetInt("page_size")
	page.PageNum, _ = self.GetInt("page_num")
	if !page.Verify() {
		page = nil
	}

	rsp := new(cydex.QueryZoneListRsp)
	rsp.Error = cydex.OK

	defer func() {
		if rsp.Zones == nil {
			rsp.Zones = make([]*cydex.Zone, 0)
		}
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	zones_m, err := models.GetZones(page)
	if err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}

	total, err := models.CountZones()
	if err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	rsp.TotalNum = int(total)
	for _, z := range zones_m {
		zone := getSignalZone(z)
		if zone != nil {
			rsp.Zones = append(rsp.Zones, zone)
		}
	}
}

func (self *ZonesController) Post() {
	req := new(cydex.Zone)
	rsp := new(cydex.ZoneRsp)

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	if self.UserLevel == cydex.USER_LEVEL_ADMIN {
		clog.Error("admin not allowed to create pkg")
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	// 获取请求
	if err := self.FetchJsonBody(req); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}
	if req.Name == "" {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	var desc string
	if req.Desc != nil {
		desc = *req.Desc
	}
	zone_m, err := models.CreateZone(uuid.New(), req.Name, desc)
	if err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	rsp.Zone = getSignalZone(zone_m)
}

type ZoneController struct {
	BaseController
}

func (self *ZoneController) Get() {
	rsp := new(cydex.ZoneRsp)
	rsp.Error = cydex.OK

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	zone_id := self.GetString(":id")
	zone_m, _ := models.GetZone(zone_id)
	if zone_m == nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	rsp.Zone = getSignalZone(zone_m)
}

func (self *ZoneController) Put() {
	self.Patch()
}

func (self *ZoneController) Patch() {
	req := new(cydex.Zone)
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

	zid := self.GetString(":id")
	// 获取请求
	if err := self.FetchJsonBody(req); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}
	if req.Name == "" {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	zone_m, _ := models.GetZone(zid)
	if zone_m == nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	if err := zone_m.SetName(req.Name); err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	if req.Desc != nil {
		if err := zone_m.SetDesc(*req.Desc); err != nil {
			rsp.Error = cydex.ErrInnerServer
			return
		}
	}
}

func (self *ZoneController) Delete() {
	rsp := new(cydex.BaseRsp)

	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	zid := self.GetString(":id")
	if err := models.DeleteZone(zid); err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	return
}

type ZoneNodesController struct {
	BaseController
}

func (self *ZoneNodesController) Put() {
	req := new(cydex.ZoneNodeRelationship)
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

	zid := self.GetString(":id")
	// 获取请求
	if err := self.FetchJsonBody(req); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	zone_m, _ := models.GetZone(zid)
	if zone_m == nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}
	if err := zone_m.AddNodes(req.AddList); err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	if err := zone_m.DelNodes(req.DelList); err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
}

func (self *ZoneNodesController) Delete() {
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

	zid := self.GetString(":id")
	zone_m, _ := models.GetZone(zid)
	if zone_m == nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}
	if err := zone_m.ClearNodes(); err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
}

func getSignalZone(zone_m *models.Zone) *cydex.Zone {
	zone := new(cydex.Zone)

	zone.Id = zone_m.Zid
	zone.Name = zone_m.Name
	zone.Desc = &zone_m.Desc
	return zone
}
