package task

import (
	"container/list"
	"errors"
	"sync"
)

// 轮询分配器
type RoundRobinScheduler struct {
	lock       sync.Mutex
	unit_queue *list.List
}

func NewRoundRobinScheduler() *RoundRobinScheduler {
	n := new(RoundRobinScheduler)
	n.unit_queue = list.New()
	return n
}

func (self *RoundRobinScheduler) Get(filter ScheduleUnitFilter, args ...interface{}) (u ScheduleUnit, err error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	found := false
	for e := self.unit_queue.Front(); e != nil; e = e.Next() {
		u = e.Value.(ScheduleUnit)
		if filter == nil {
			found = true
		} else {
			if filter(u, args...) {
				found = true
			}
		}
		if found {
			self.unit_queue.MoveToBack(e) // 放在队尾
			break
		}
	}
	if !found {
		u = nil
		err = errors.New("No valid unit to schedule")
	}
	return
}

func (self *RoundRobinScheduler) Put(u ScheduleUnit) {
	if u == nil {
		return
	}
	self.lock.Lock()
	defer self.lock.Unlock()
	self.unit_queue.PushBack(u)
}

func (self *RoundRobinScheduler) Pop(u ScheduleUnit) {
	if u == nil {
		return
	}
	self.lock.Lock()
	defer self.lock.Unlock()

	for e := self.unit_queue.Front(); e != nil; e = e.Next() {
		if e.Value == u {
			self.unit_queue.Remove(e)
			break
		}
	}
}

func (self *RoundRobinScheduler) NumUnits() int {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.unit_queue.Len()
}

func (self *RoundRobinScheduler) Reset() {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.unit_queue = self.unit_queue.Init()
}
