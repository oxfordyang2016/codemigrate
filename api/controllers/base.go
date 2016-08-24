package controllers

import (
	"encoding/json"
	"github.com/astaxie/beego"
	clog "github.com/cihub/seelog"
	"io/ioutil"
	"strconv"
)

var (
	ShowReqBody = false
	ShowRspBody = false
)

type BaseController struct {
	beego.Controller
	UserLevel int
}

func (self *BaseController) fetchUserLevel() {
	ul := self.Ctx.Input.Header("x-cydex-userlevel")
	if ul != "" {
		i, err := strconv.Atoi(ul)
		if err == nil {
			self.UserLevel = i
		}
	}
}

func (self *BaseController) Prepare() {
	self.fetchUserLevel()
}

func (self *BaseController) Finish() {
	if ShowRspBody && self.Data["json"] != nil {
		b, err := json.Marshal(self.Data["json"])
		if err == nil {
			clog.Trace("[http rsp]: ", string(b))
		}
	}
}

func (self *BaseController) FetchJsonBody(v interface{}) error {
	r := self.Ctx.Request
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		clog.Error(err)
		return err
	}
	if ShowReqBody {
		clog.Trace("[http req]: ", string(body))
	}
	if err := json.Unmarshal(body, v); err != nil {
		clog.Error(err)
		return err
	}
	return nil
}

func SetupBodyShow(req, rsp bool) {
	ShowReqBody = req
	ShowRspBody = rsp
}
