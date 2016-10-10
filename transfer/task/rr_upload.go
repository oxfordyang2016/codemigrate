package task

import (
	trans "./.."
	"cydex/transfer"
)

// 按照轮询方式分配的上传任务分配器
type RoundRobinUploadScheduler struct {
	rr *RoundRobinScheduler
}

func NewRoundRobinUploadScheduler() *RoundRobinUploadScheduler {
	n := new(RoundRobinUploadScheduler)
	n.rr = NewRoundRobinScheduler()
	return n
}

func filteUnitBySize(u ScheduleUnit, args ...interface{}) bool {
	arg := args[0]
	if arg == nil {
		return true
	}

	req := arg.(*UploadReq)
	if u.FreeStorage() >= GetReqSize(req) {
		return true
	}

	return false
}

func (self *RoundRobinUploadScheduler) DispatchUpload(req *UploadReq) (n *trans.Node, err error) {
	unit, err := self.rr.Get(filteUnitBySize, req)
	if err != nil {
		return nil, err
	}
	n = unit.GetNode()
	return
}

func (self *RoundRobinUploadScheduler) AddNode(n *trans.Node) {
	// 新增node放队尾
	self.rr.Put(n)
}

func (self *RoundRobinUploadScheduler) DelNode(n *trans.Node) {
	// 删除失去连接的node
	self.rr.Pop(n)
}

func (self *RoundRobinUploadScheduler) UpdateNode(n *trans.Node, req *transfer.KeepaliveReq) {
	// Do nothing
}

func (self *RoundRobinUploadScheduler) NumUnits() int {
	return self.rr.NumUnits()
}

func (self *RoundRobinUploadScheduler) Reset() {
	self.rr.Reset()
}
