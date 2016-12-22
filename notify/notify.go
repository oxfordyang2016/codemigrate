package notify

const (
	NotifyToReceiverNewPkg            = 1 // 通知接收者，有新包需要下载
	NotifyToSenderPkgUploadFinish     = 2 // 通知发送者，上传完毕
	NotifyToReceiverPkgDownloadFinish = 3 // 通知接收者，下载完毕
	NotifyToSenderPkgDownloadStart    = 4 // 通知发送者，某个下载者开始下载
	NotifyToSenderPkgDownloadFinish   = 5 // 通知发送者，某个下载者下载完毕
)

// 通知事件
type Event struct {
	typ int
	// TODO 记录job的元信息，包裹的名称，事件，发送者，接收者等, 供Handler处理
}

// 通知
type Handler interface {
	HandleEvent(e *Event) error
}

type NotifyManage struct {
	handlers []Handler
}

func (self *NotifyManage) AddHandler(h Handler) {
	if h == nil {
		return
	}
	self.handlers = append(self.handlers, h)
}

// implement pkg.JobObserver
func (self *NotifyManage) OnJobCreate() {
}

func (self *NotifyManage) OnJobStart() {
}

func (self *NotifyManage) OnJobFinish() {
}
