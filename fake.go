package main

import (
	api_ctrl "./api/controllers"
	"github.com/astaxie/beego"
	clog "github.com/cihub/seelog"
)

// jzh: 这个是为了能单独跑起测试用的,通过ts.ini的[http]:[fake_api]项可以开启或者关闭

type RootController struct {
	beego.Controller
}

func (self *RootController) Get() {
	query := self.GetString("query")

	if query == "alias" {
		self.Ctx.Output.Body([]byte(`{"error":0, "alias":"fake_server"}`))
	}
}

type UserController struct {
	beego.Controller
}

func (self *UserController) Get() {
	query := self.GetString("query")
	if query == "availuser" {
		json := `{
			"error":0,
			"user":[{
				"username": "receiver", 
				"uid": "rr00000000rr", 
				"state": 1, 
				"expire": null, 
				"last_login": "2016-07-29 03:07:21", 
				"u_level": 0
			}]
		}`

		self.Ctx.Output.Body([]byte(json))
	}
}

type LoginController struct {
	api_ctrl.BaseController
}

func (self *LoginController) Post() {
	type LoginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	req := new(LoginReq)

	if err := self.FetchJsonBody(req); err != nil {
		return
	}
	// 模拟多个用户
	result := make(map[string]string)
	result["sender"] = `{"usr": {"username": "sender", "uid": "ss00000000ss", "state": 1, "expire": null, "last_login": "2016-07-29 03:07:21", "u_level": 0}, "session_id": "xyacqrbo5z3kqqf3hadovsio0gz7407q", "error": 0}`
	result["receiver"] = `{"usr": {"username": "receiver", "uid": "rr00000000rr", "state": 1, "expire": null, "last_login": "2016-07-29 03:07:21", "u_level": 0}, "session_id": "zyacqrbo5z3kqqf3hadovsio0gz7407q", "error": 0}`

	// 返回结果
	self.Ctx.Output.Body([]byte(result[req.Username]))
}

type TestController struct {
	api_ctrl.BaseController
}

func (self *TestController) Get() {
	clog.Tracef("user level:%d", self.UserLevel)
	self.Ctx.WriteString("this is test")
}

func EnableFakeApi() {
	beego.Router("/", &RootController{})
	beego.Router("/:uid/user", &UserController{})
	beego.Router("/login", &LoginController{})

	beego.Router("/test", &TestController{})
}
