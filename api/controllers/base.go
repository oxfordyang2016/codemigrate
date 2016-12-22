package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego"
	clog "github.com/cihub/seelog"
	"io/ioutil"
	"net"
	"strconv"
	"time"
)

var (
	ShowReqBody = false
	ShowRspBody = false
)

type BaseController struct {
	beego.Controller
	UserLevel              int
	Ip                     string
	UserMaxUploadBitrate   uint64
	UserMaxDownloadBitrate uint64
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

func (self *BaseController) fetchIp() {
	ip := self.Ctx.Input.Header("x-cydex-ip")
	self.Ip = ip
}

func (self *BaseController) fetchUserMaxBitrate() {
	v := self.Ctx.Input.Header("x-cydex-user-max-upload-bitrate")
	if v != "" {
		i, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			self.UserMaxUploadBitrate = i
		}
	}
	v = self.Ctx.Input.Header("x-cydex-user-max-download-bitrate")
	if v != "" {
		i, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			self.UserMaxDownloadBitrate = i
		}
	}
}

func (self *BaseController) Prepare() {
	self.fetchUserLevel()
	self.fetchIp()
	self.fetchUserMaxBitrate()
}

func (self *BaseController) Finish() {
	if ShowRspBody && self.Data["json"] != nil {
		b, err := json.Marshal(self.Data["json"])
		if err == nil {
			show_b := b
			if len(b) > 256 {
				show_b = b[:256]
			}
			clog.Tracef("[http rsp]: %s ...", string(show_b))
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

func MarshalUTCTime(t time.Time) string {
	utc := t.UTC()
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", utc.Year(), utc.Month(), utc.Day(), utc.Hour(), utc.Minute(), utc.Second())
}

func verifyIPAddr(addr string) bool {
	ip := net.ParseIP(addr)
	return ip != nil
}
