package task

import (
	trans "./.."
	clog "github.com/cihub/seelog"
	URL "net/url"
)

// 按照"file://"类型存储的下载任务分配器
type FileDownloadScheduler struct {
}

func NewFileDownloadScheduler() *FileDownloadScheduler {
	n := new(FileDownloadScheduler)
	return n
}

func (self *FileDownloadScheduler) DispatchDownload(req *DownloadReq) (n *trans.Node, err error) {
	clog.Trace("in file dispatch download")
	url, ok := req.meta.(*URL.URL)
	if !ok {
		clog.Error("meta to url failed")
		return nil, err
	}
	clog.Tracef("%#v", url)
	nid := url.Host
	clog.Info(nid)
	n = trans.NodeMgr.GetByNid(nid)
	return
}
