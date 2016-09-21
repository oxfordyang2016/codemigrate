package main

import (
	_ "./api/"
	api_ctrl "./api/controllers"
	"./pkg"
	pkg_model "./pkg/models"
	trans "./transfer"
	trans_model "./transfer/models"
	"./transfer/task"
	"./utils/cache"
	"./utils/db"
	"fmt"
	"github.com/astaxie/beego"
	clog "github.com/cihub/seelog"
	"github.com/go-ini/ini"
	"time"
)

const (
	VERSION = "0.0.1-alpha1"
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
	if err = pkg_model.SyncTables(); err != nil {
		return
	}
	if err = trans_model.SyncTables(); err != nil {
		return
	}
	return
}

func setupCache(cfg *ini.File) (err error) {
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

	var unpacker pkg.Unpacker
	unpacker_str := sec.Key("unpacker").MustString("default")
	clog.Infof("[setup pkg] unpacker: %s", unpacker_str)

	switch unpacker_str {
	case "default":
		sec_unpacker, err := cfg.GetSection("default_unpacker")
		if err != nil {
			return err
		}
		min_seg_size := ConfigGetSize(sec_unpacker.Key("min_seg_size").String())
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
		clog.Warnf("Enable fake api for standalone running")
		EnableFakeApi()
	}

	beego_loglevel = sec.Key("beego_loglevel").MustInt(beego.LevelInformational)
	return
}

func setupApplication(cfg *ini.File) (err error) {
	if err = setupDB(cfg); err != nil {
		return
	}
	if err = setupCache(cfg); err != nil {
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
	return
}

func run() {
	// setup version
	ws_server.SetVersion(VERSION)
	// add job listen
	task.TaskMgr.AddObserver(pkg.JobMgr)
	// 从数据库导入track
	pkg.JobMgr.LoadTracks()

	go task.TaskMgr.TaskStateRoutine()
	go ws_server.Serve()

	beego.SetLevel(beego_loglevel)
	go beego.Run(http_addr)

	clog.Info("Run...")
}

func main() {
	initLog()
	clog.Infof("Start cydex transfer service, version:%s", VERSION)

	cfg, err := LoadConfig()
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
