package task

import (
	trans "./.."
	"cydex/transfer"
	"errors"
)

// 按照轮询方式分配的上传任务分配器
type RoundRobinUploadScheduler struct {
	*RoundRobinScheduler
}

func NewRoundRobinUploadScheduler() *RoundRobinUploadScheduler {
	n := new(RoundRobinUploadScheduler)
	n.RoundRobinScheduler = NewRoundRobinScheduler()
	return n
}

func filteUnitBySize(u ScheduleUnit, args ...interface{}) bool {
	req := args[0].(*UploadReq)
	if req == nil {
		return true
	}

	node, ok := u.(*trans.Node)
	if !ok {
		return false
	}
	if node.Info.FreeStorage >= GetReqSize(req) {
		return true
	}

	return false
}

func (self *RoundRobinUploadScheduler) DispatchUpload(req *UploadReq) (n *trans.Node, err error) {
	unit, err := self.Get(filteUnitBySize, req)
	if err != nil {
		return nil, err
	}
	var ok bool
	n, ok = unit.(*trans.Node)
	if !ok {
		n = nil
		err = errors.New("ScheduleUnit is not node")
	}
	return
}

func (self *RoundRobinUploadScheduler) AddNode(n *trans.Node) {
	// 新增node放队尾
	self.Put(n)
}

func (self *RoundRobinUploadScheduler) DelNode(n *trans.Node) {
	// 删除失去连接的node
	self.Pop(n)
}

func (self *RoundRobinUploadScheduler) UpdateNode(n *trans.Node, req *transfer.KeepaliveReq) {
	// Do nothing
}

func (self *RoundRobinUploadScheduler) NodeZoneChange(n *trans.Node, old_zid, new_zid string) {
	// Do nothing
}
