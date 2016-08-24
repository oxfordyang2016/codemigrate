package task

import (
	trans "./.."
	"fmt"
	clog "github.com/cihub/seelog"
	URL "net/url"
	"strings"
)

// 上传调度器接口
type UploadScheduler interface {
	DispatchUpload(req *UploadReq) (*trans.Node, error)
}

// 下载调度器接口
type DownloadScheduler interface {
	DispatchDownload(req *DownloadReq) (*trans.Node, error)
}

// 任务调度器接口
type TaskScheduler interface {
	String() string
	UploadScheduler
	DownloadScheduler
}

// 默认的任务调度器
type DefaultScheduler struct {
	u_restrict *RestrictUploadScheduler
	u_proxy    UploadScheduler
	d_file     *FileDownloadScheduler
	d_nas      *NasDownloadScheduler
}

func NewDefaultScheduler() *DefaultScheduler {
	n := new(DefaultScheduler)
	rr := NewRoundRobinUploadScheduler()
	n.u_proxy = rr
	n.u_restrict = NewRestrictUploadScheduler(TASK_RESTRICT_BY_PID)
	n.d_file = NewFileDownloadScheduler()
	n.d_nas = NewNasDownloadScheduler()

	trans.NodeMgr.AddObserver(rr)
	TaskMgr.AddObserver(n.u_restrict)
	return n
}

func (self *DefaultScheduler) String() string {
	return "Default Scheduler"
}

func (self *DefaultScheduler) DispatchUpload(req *UploadReq) (n *trans.Node, err error) {
	// 首先使用约束调度器, 如果没有找到,则再使用具体的上传调度器进行分配
	clog.Tracef("%s: dispatch upload", self)
	n, err = self.u_restrict.DispatchUpload(req)
	if err != nil {
		// log
	}
	// var first_node *trans.Node
	if n == nil {
		n, err = self.u_proxy.DispatchUpload(req)
	}
	if err != nil {
		// log
	}
	return
}

func (self *DefaultScheduler) DispatchDownload(req *DownloadReq) (n *trans.Node, err error) {
	clog.Trace("dispatch download")
	// TODO: 客户端一次请求一个file的所有seg,上传也是, 所以这些seg的分段存在一个Node上
	// 这里判断storage是否是一个nodeid上?

	// 取第一个即可
	sid_storage := req.SidStorage[0]
	if sid_storage == "" {
		err = fmt.Errorf("segs are not uploaded yet!")
		return
	}
	clog.Trace(sid_storage)
	var url *URL.URL
	if url, err = URL.Parse(sid_storage); err != nil {
		return nil, err
	}
	scheme := strings.ToLower(url.Scheme)
	clog.Trace(scheme)
	switch scheme {
	case "file":
		req.meta = url
		n, err = self.d_file.DispatchDownload(req)
	case "nas":
		n, err = self.d_nas.DispatchDownload(req)
	default:
		err = fmt.Errorf("Unsupport storage url %s", sid_storage)
	}
	return
}
