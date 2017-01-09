package main

import (
	_ "./api/"
	api_ctrl "./api/controllers"
	"./pkg"
	pkg_model "./pkg/models"
	// "./statistics"
	// stat_model "./statistics/models"
	"./notify"
	trans "./transfer"
	trans_model "./transfer/models"
	"./transfer/task"
	"./utils"
	"./utils/cache"
	"./utils/db"
	"fmt"
	"github.com/astaxie/beego"
	clog "github.com/cihub/seelog"
	"github.com/go-ini/ini"
	"time"
)

const (
	VERSION = "0.2.0"
)

const (
	PROFILE = "/opt/cydex/etc/ts.d/profile.ini"
	CFGFILE = "/opt/cydex/config/ts.ini"
)

var (
	ws_server      *trans.WSServer
	http_addr      string
	beego_loglevel int
)

func initLog() {
	cfgfiles := []string{
		"/opt/cydex/etc/ts.d/seelog.xml",
		"seelog.xml",
	}
	for _, file := range cfgfiles {
		logger, err := clog.LoggerFromConfigAsFile(file)
		if err != nil {
			println(err.Error())
			continue
		}
		clog.ReplaceLogger(logger)
		break
	}
}

func setupUserDB(cfg *ini.File) (err error) {
	// 连接User数据库
	clog.Info("setup user db")
	sec, err := cfg.GetSection("user_db")
	if err != nil {
		return err
	}
	driver := sec.Key("driver").String()
	source := sec.Key("source").String()
	show_sql := sec.Key("show_sql").MustBool()
	clog.Infof("[setup db] url:'%s %s' show_sql:%t", driver, source, show_sql)
	if err = db.CreateUserEngine(driver, source, show_sql); err != nil {
		return
	}

	return
}

func setupDB(cfg *ini.File) (err error) {
	// 创建数据库
	clog.Info("setup db")
	sec, err := cfg.GetSection("db")
	if err != nil {
		return err
	}
	driver := sec.Key("driver").String()
	source := sec.Key("source").String()
	show_sql := sec.Key("show_sql").MustBool()
	clog.Infof("[setup db] url:'%s %s' show_sql:%t", driver, source, show_sql)
	if err = db.CreateEngine(driver, source, show_sql); err != nil {
		return
	}
	// 同步表结构
	var tables []interface{}
	tables = append(tables, pkg_model.Tables...)
	tables = append(tables, trans_model.Tables...)
	// tables = append(tables, stat_model.Tables...)
	if err = db.SyncTables(tables); err != nil {
		return
	}

	// 设置cache
	var caches []interface{}
	caches = append(caches, pkg_model.Caches...)
	caches = append(caches, trans_model.Caches...)
	// caches = append(caches, stat_model.Caches...)
	db.MapCache(caches)

	// migrate db and data
	clog.Infof("[setup db] migrate...")
	pkg.JobMgr.JobsSyncState()
	clog.Infof("[setup db] migrate ok")

	return
}

func setupRedis(cfg *ini.File) (err error) {
	sec, err := cfg.GetSection("redis")
	if err != nil {
		return err
	}
	url := sec.Key("url").String()
	max_idles := sec.Key("max_idles").MustInt(3)
	idle_timeout := sec.Key("idle_timeout").MustInt(240)
	clog.Infof("[setup cache] url:'%s' max_idle:%d idle_timeout:%d", url, max_idles, idle_timeout)
	cache.Init(url, max_idles, idle_timeout)
	// test cache first
	err = cache.Ping()
	if err != nil {
		return err
	}
	return
}

func setupWebsocketService(cfg *ini.File) (err error) {
	sec, err := cfg.GetSection("websocket")
	if err != nil {
		return err
	}
	url := sec.Key("url").String()
	port := sec.Key("port").RangeInt(-1, 0, 65535)
	if url == "" || port == -1 {
		err = fmt.Errorf("Invalid websocket url:%s or port:%d", url, port)
		return
	}
	ssl := sec.Key("ssl").MustBool(false)
	cert_file := sec.Key("cert_file").MustString("")
	key_file := sec.Key("key_file").MustString("")
	idle_timeout := sec.Key("idle_timeout").MustUint(10)
	keepalive_interval := sec.Key("keepalive_interval").MustUint(180)
	notify_interval := sec.Key("notify_interval").MustUint(3)
	ws_cfg := &trans.WSServerConfig{
		UseSSL:                 ssl,
		CertFile:               cert_file,
		KeyFile:                key_file,
		ConnDeadline:           time.Duration(idle_timeout) * time.Second,
		KeepaliveInterval:      keepalive_interval,
		TransferNotifyInterval: notify_interval,
	}
	clog.Infof("[setup websocket] listen port:%d, ssl:%t with config [%d, %d, %d]", port, ssl, idle_timeout, keepalive_interval, notify_interval)
	ws_server = trans.NewWSServer(url, port, ws_cfg)
	return
}

func setupPkg(cfg *ini.File) (err error) {
	sec, err := cfg.GetSection("pkg")
	if err != nil {
		return err
	}

	file_slice := sec.Key("file_slice").MustBool(true)
	pkg.SetUsingFileSlice(file_slice)

	var unpacker pkg.Unpacker
	unpacker_str := sec.Key("unpacker").MustString("default")
	clog.Infof("[setup pkg] unpacker: %s, file slice: %t", unpacker_str, file_slice)

	switch unpacker_str {
	case "default":
		sec_unpacker, err := cfg.GetSection("default_unpacker")
		if err != nil {
			return err
		}
		min_seg_size := utils.ConfigGetSize(sec_unpacker.Key("min_seg_size").String())
		if min_seg_size == 0 {
			err = fmt.Errorf("Invalid min_seg_size:%d", min_seg_size)
			return err
		}
		max_seg_cnt, err := sec_unpacker.Key("max_seg_cnt").Uint()
		if err != nil {
			return err
		}
		clog.Infof("[setup pkg] default unpacker: %d, %d", min_seg_size, max_seg_cnt)
		unpacker = pkg.NewDefaultUnpacker(uint64(min_seg_size), max_seg_cnt)
	default:
		err = fmt.Errorf("Unsupport unpacker name:%s", unpacker)
		return
	}
	pkg.SetUnpacker(unpacker)
	return
}

func setupTask(cfg *ini.File) (err error) {
	sec, err := cfg.GetSection("task")
	if err != nil {
		return err
	}

	var restrict_mode int
	restrict_str := sec.Key("restrict_mode").MustString("pid")
	clog.Infof("[setup task] restrict_mode: %s", restrict_str)
	switch restrict_str {
	case "pid":
		restrict_mode = task.TASK_RESTRICT_BY_PID
	case "fid":
		restrict_mode = task.TASK_RESTRICT_BY_FID
	default:
		err = fmt.Errorf("Unknow restrict mode %s", restrict_str)
		return
	}

	scheduler_str := sec.Key("scheduler").MustString("default")
	clog.Infof("[setup task] scheduler:%s", scheduler_str)
	switch scheduler_str {
	case "default":
		task.TaskMgr.SetScheduler(task.NewDefaultScheduler(restrict_mode))
	default:
		err = fmt.Errorf("Unsupport task scheduler name:%s", scheduler_str)
		return
	}
	cache_timeout, err := sec.Key("cache_timeout").Int()
	if err != nil {
		return err
	}
	clog.Infof("[setup task] task cache timeout:%d", cache_timeout)
	task.TaskMgr.SetCacheTimeout(cache_timeout)
	return
}

func setupHttp(cfg *ini.File) (err error) {
	sec, err := cfg.GetSection("http")
	if err != nil {
		return err
	}
	addr := sec.Key("addr").String()
	if addr == "" {
		err = fmt.Errorf("Http addr can't be empty!")
		return
	}
	show_req := sec.Key("show_req").MustBool(false)
	show_rsp := sec.Key("show_rsp").MustBool(false)
	clog.Infof("[setup http] addr:'%s', show: [%t, %t]", addr, show_req, show_rsp)
	http_addr = addr
	api_ctrl.SetupBodyShow(show_req, show_rsp)

	fake_api := sec.Key("fake_api").MustBool(false)
	if fake_api {
		clog.Infof("Enable fake api for standalone running")
		EnableFakeApi()
	}

	beego_loglevel = sec.Key("beego_loglevel").MustInt(beego.LevelInformational)
	return
}

func setupNotificationEmailHandler(cfg *ini.File, sec *ini.Section) error {
	enable := sec.Key("enable").MustBool(false)
	smtp_label := sec.Key("smtp_server").MustString("default")
	contact_name := sec.Key("contact_name").String()
	templates_dir := sec.Key("templates_dir").String()

	smtp_sec, err := cfg.GetSection(fmt.Sprintf("smtp_server.%s", smtp_label))
	if err != nil {
		return err
	}

	smtp_server := notify.SmtpServer{
		Label:    smtp_label,
		Host:     smtp_sec.Key("host").String(),
		Port:     smtp_sec.Key("port").MustInt(0),
		Account:  smtp_sec.Key("account").String(),
		Password: smtp_sec.Key("password").String(),
		UseTLS:   smtp_sec.Key("use_tls").MustBool(false),
	}

	notify.Email.SetEnable(enable)
	notify.Email.SetContactName(contact_name)
	notify.Email.SetSmtpServer(&smtp_server)
	notify.Email.SetTemplateBaseDir(templates_dir)
	// load templates
	if err := notify.Email.LoadTemplates(); err != nil {
		return err
	}
	return nil
}

func setupNotification(cfg *ini.File) (err error) {
	sec, err := cfg.GetSection("notification")
	if err != nil {
		return err
	}

	enable := sec.Key("enable").MustBool(false)
	notify.Manage.SetEnable(enable)

	handlers := sec.Key("handlers").Strings(utils.CFG_SEP)
	for _, handler := range handlers {
		handler_sec, err := cfg.GetSection(fmt.Sprintf("notification.%s", handler))
		if err != nil {
			continue
		}
		switch handler {
		case "email":
			if err := setupNotificationEmailHandler(cfg, handler_sec); err != nil {
				return err
			}
		}
	}

	return nil
}

func setupApplication(cfg *ini.File) (err error) {
	if err = setupDB(cfg); err != nil {
		return
	}
	if err = setupUserDB(cfg); err != nil {
		return
	}
	if err = setupRedis(cfg); err != nil {
		return
	}
	if err = setupPkg(cfg); err != nil {
		return
	}
	if err = setupTask(cfg); err != nil {
		return
	}
	if err = setupWebsocketService(cfg); err != nil {
		return
	}
	if err = setupHttp(cfg); err != nil {
		return
	}
	if err = setupNotification(cfg); err != nil {
		return
	}
	return
}

func run() {
	// setup version
	ws_server.SetVersion(VERSION)
	// task add observer for job
	task.TaskMgr.AddObserver(pkg.JobMgr)
	// task add observer for statistics
	// task.TaskMgr.AddObserver(statistics.TransferMgr)
	// 从数据库导入track
	pkg.JobMgr.LoadTracks()
	// job add observer for notification
	pkg.JobMgr.AddJobObserver(notify.Manage)

	go task.TaskMgr.TaskStateRoutine()
	go ws_server.Serve()

	clog.Info("Notification start")
	go notify.Manage.Serve()

	clog.Info("Notification email handler start")
	notify.Email.Start()

	beego.SetLevel(beego_loglevel)
	go beego.Run(http_addr)

	clog.Info("Run...")
}

func main() {
	initLog()
	clog.Infof("Start cydex transfer service, version:%s", VERSION)

	config := utils.NewConfig(PROFILE, CFGFILE)
	if config == nil {
		clog.Critical("new config failed")
		return
	}
	utils.MakeDefaultConfig(config)
	cfg, err := config.Load()
	if err != nil {
		clog.Critical(err)
		return
	}

	if err = setupApplication(cfg); err != nil {
		clog.Critical(err)
		return
	}

	run()

	for {
		time.Sleep(10 * time.Minute)
	}
}
