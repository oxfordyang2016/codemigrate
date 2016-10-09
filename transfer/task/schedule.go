package task

import (
	trans "./.."
	// "cydex/transfer"
	// "fmt"
	// clog "github.com/cihub/seelog"
	// URL "net/url"
	// "strings"
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

// 调度单元, node or zone
type ScheduleUnit interface {
	// Id() string
	FreeStorage() uint64
	UploadScheduler
	DownloadScheduler
}
