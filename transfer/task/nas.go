package task

import (
	trans "./.."
)

// 按照"nas://"类型存储的下载任务分配器, 需要实现NodeObserver接口
type NasDownloadScheduler struct {
}

func NewNasDownloadScheduler() *NasDownloadScheduler {
	n := new(NasDownloadScheduler)
	return n
}

func (self *NasDownloadScheduler) DispatchDownload(req *DownloadReq) (n *trans.Node, err error) {
	return
}
