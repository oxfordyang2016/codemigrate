package task

import (
	trans "./.."
)

// TODO
// 按照"file://"类型存储的下载任务分配器
type FileDownloadScheduler struct {
}

func NewFileDownloadScheduler() *FileDownloadScheduler {
	n := new(FileDownloadScheduler)
	return n
}

func (self *FileDownloadScheduler) DispatchDownload(req *DownloadReq) (n *trans.Node, err error) {
	return
}
