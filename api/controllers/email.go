package controllers

import (
	"./../../notify"
	"./../../utils"
	"cydex"
	"fmt"
	clog "github.com/cihub/seelog"
)

type EmailController struct {
	BaseController
}

func (self *EmailController) Get() {
	rsp := new(cydex.GetEmailInfoRsp)
	rsp.Error = cydex.OK

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	var label string
	smtp_server := notify.Email.SmtpServer()
	if smtp_server != nil {
		label = smtp_server.Label
	}

	rsp.Email = &cydex.EmailInfo{
		Enable:      &notify.Email.Enable,
		ContactName: &notify.Email.ContactName,
		SmtpServer:  &label,
	}
}

func (self *EmailController) Put() {
	req := new(cydex.EmailInfo)
	rsp := new(cydex.GetEmailInfoRsp)
	rsp.Error = cydex.OK

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	// 获取请求
	if err := self.FetchJsonBody(req); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	// setup
	if req.Enable != nil {
		notify.Email.SetEnable(*req.Enable)
	}
	if req.ContactName != nil {
		notify.Email.SetContactName(*req.ContactName)
	}

	var label string
	smtp_server := notify.Email.SmtpServer()
	if smtp_server != nil {
		label = smtp_server.Label
	}
	if req.SmtpServer != nil && *req.SmtpServer != label {
		// 重新获取配置
		smtp_label := *req.SmtpServer
		cfg, err := utils.DefaultConfig().Load()
		if err != nil {
			rsp.Error = cydex.ErrInnerServer
			return
		}
		smtp_sec, err := cfg.GetSection(fmt.Sprintf("smtp_server.%s", smtp_label))
		if err != nil {
			rsp.Error = cydex.ErrInvalidParam
			return
		}

		smtp_server := notify.SmtpServer{
			Label:    smtp_label,
			Host:     smtp_sec.Key("host").String(),
			Port:     smtp_sec.Key("port").MustInt(0),
			Account:  smtp_sec.Key("account").String(),
			Password: smtp_sec.Key("password").String(),
			UseTLS:   smtp_sec.Key("use_tls").MustBool(false),
		}
		if err := notify.Email.SetSmtpServer(&smtp_server); err != nil {
			rsp.Error = cydex.ErrInnerServer
			return
		}
	}
	// save
	if err := utils.DefaultConfig().SaveEmailInfo(req); err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
}

func (self *EmailController) Post() {
	self.Put()
}

type EmailTemplatesController struct {
	BaseController
}

func (self *EmailTemplatesController) Post() {
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

	if err := notify.Email.LoadTemplates(); err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
}

type EmailSmtpsController struct {
	BaseController
}

func (self *EmailSmtpsController) Get() {
	rsp := new(cydex.GetEmailSmtpServerListRsp)
	rsp.Error = cydex.OK

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	cfg, err := utils.DefaultConfig().Load()
	if err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}
	for _, label := range notify.SmtpLabels {
		smtp_server := new(cydex.EmailSmtpServer)
		smtp_server.Label = label
		rsp.SmtpServerList = append(rsp.SmtpServerList, smtp_server)
		sec, err := cfg.GetSection(fmt.Sprintf("smtp_server.%s", label))
		if err != nil {
			clog.Error(err)
			continue
		}
		smtp_server.Host = sec.Key("host").String()
		smtp_server.Port = sec.Key("port").MustInt(0)
		smtp_server.Account = sec.Key("account").String()
		smtp_server.Password = sec.Key("password").String()
		smtp_server.UseTLS = sec.Key("use_tls").MustBool(false)
	}
}

type EmailSmtpController struct {
	BaseController
}

func (self *EmailSmtpController) Get() {
	rsp := new(cydex.GetEmailSmtpServerRsp)
	rsp.Error = cydex.OK

	defer func() {
		self.Data["json"] = rsp
		self.ServeJSON()
	}()

	if self.UserLevel != cydex.USER_LEVEL_ADMIN {
		rsp.Error = cydex.ErrNotAllowed
		return
	}

	label := self.GetString(":label")
	found := false
	for _, l := range notify.SmtpLabels {
		if label == l {
			found = true
			break
		}
	}
	if !found {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	cfg, err := utils.DefaultConfig().Load()
	if err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}

	rsp.SmtpServer = &cydex.EmailSmtpServer{
		Label: label,
	}
	sec, err := cfg.GetSection(fmt.Sprintf("smtp_server.%s", label))
	if err != nil {
		return
	}
	rsp.SmtpServer.Host = sec.Key("host").String()
	rsp.SmtpServer.Port = sec.Key("port").MustInt(0)
	rsp.SmtpServer.Account = sec.Key("account").String()
	rsp.SmtpServer.Password = sec.Key("password").String()
	rsp.SmtpServer.UseTLS = sec.Key("use_tls").MustBool(false)
}

func (self *EmailSmtpController) Post() {
	req := new(cydex.EmailSmtpServer)
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

	label := self.GetString(":label")
	found := false
	for _, l := range notify.SmtpLabels {
		if label == l {
			found = true
			break
		}
	}
	if !found || label == "default" {
		rsp.Error = cydex.ErrInvalidParam
		return
	}

	// 获取请求
	if err := self.FetchJsonBody(req); err != nil {
		rsp.Error = cydex.ErrInvalidParam
		return
	}
	req.Label = label

	if err := utils.DefaultConfig().SaveEmailSmtpServer(req, label); err != nil {
		rsp.Error = cydex.ErrInnerServer
		return
	}

	var cur_label string
	smtp_server := notify.Email.SmtpServer()
	if smtp_server != nil {
		cur_label = smtp_server.Label
	}
	// jzh: 如果EmailHandler的当前smtp_server是修改的这个，需要重新设置一下
	if cur_label == label {
		clog.Infof("The smtp server is being used by email handler, should be setup!")
		if err := notify.Email.SetSmtpServer((*notify.SmtpServer)(req)); err != nil {
			clog.Errorf("Set smtp server failed after modified the smtp config, %s", err.Error())
			rsp.Error = cydex.ErrInnerServer
			return
		}
	}
}
