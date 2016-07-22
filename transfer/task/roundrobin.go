package task

import (
	trans "./.."
	"container/list"
	"cydex/transfer"
	"errors"
	"sync"
)

// 按照轮询方式分配的上传任务分配器
type RoundRobinUploadScheduler struct {
	lock       sync.Mutex
	node_queue *list.List
}

func NewRoundRobinUploadScheduler() *RoundRobinUploadScheduler {
	n := new(RoundRobinUploadScheduler)
	n.node_queue = list.New()
	return n
}

func (self *RoundRobinUploadScheduler) DispatchUpload(req *UploadReq) (n *trans.Node, err error) {
	defer self.lock.Unlock()
	self.lock.Lock()

	found := false
	for e := self.node_queue.Front(); e != nil; e = e.Next() {
		n = e.Value.(*trans.Node)
		if req == nil {
			found = true
		} else {
			if n.Info.FreeStorage >= req.Size {
				found = true
			}
		}
		if found {
			self.node_queue.MoveToBack(e) // 放在队尾
			break
		}
	}
	if !found {
		n = nil
		err = errors.New("No found")
	}
	return
}

func (self *RoundRobinUploadScheduler) AddNode(n *trans.Node) {
	// 新增node放队尾
	defer self.lock.Unlock()
	self.lock.Lock()
	self.node_queue.PushBack(n)
}

func (self *RoundRobinUploadScheduler) DelNode(n *trans.Node) {
	// 删除失去连接的node
	defer self.lock.Unlock()
	self.lock.Lock()

	for e := self.node_queue.Front(); e != nil; e = e.Next() {
		if e.Value == n {
			self.node_queue.Remove(e)
			break
		}
	}
}

func (self *RoundRobinUploadScheduler) UpdateNode(n *trans.Node, req *transfer.KeepaliveReq) {
	// Do nothing
}

func (self *RoundRobinUploadScheduler) NumNodes() int {
	defer self.lock.Unlock()
	self.lock.Lock()
	return self.node_queue.Len()
}

func (self *RoundRobinUploadScheduler) Reset() {
	defer self.lock.Unlock()
	self.lock.Lock()
	self.node_queue = self.node_queue.Init()
}
