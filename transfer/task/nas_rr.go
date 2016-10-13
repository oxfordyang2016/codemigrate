package task

import (
	trans "./.."
	"cydex/transfer"
	"fmt"
	clog "github.com/cihub/seelog"
	"sync"
)

// zone下有多个node, 这里实现的是RoundRobin调度

type NasRoundRobinScheduler struct {
	mux        sync.Mutex
	zone_schds map[string]*RoundRobinScheduler
}

func NewNasRoundRobinScheduler() *NasRoundRobinScheduler {
	o := new(NasRoundRobinScheduler)
	o.zone_schds = make(map[string]*RoundRobinScheduler)
	return o
}

// implement NodeObserver
func (self *NasRoundRobinScheduler) AddNode(n *trans.Node) {
	// 新增node
	if n.ZoneId == "" {
		return
	}
	self.mux.Lock()
	defer self.mux.Unlock()
	self.zoneAddNode(n.ZoneId, n)
}

func (self *NasRoundRobinScheduler) zoneAddNode(zid string, n *trans.Node) {
	if zid == "" {
		return
	}
	zone_schd, ok := self.zone_schds[zid]
	if !ok {
		zone_schd = NewRoundRobinScheduler()
		self.zone_schds[zid] = zone_schd
	}
	zone_schd.Put(n)
}

func (self *NasRoundRobinScheduler) zoneDelNode(zid string, n *trans.Node) {
	zone_schd, ok := self.zone_schds[zid]
	if !ok {
		return
	}
	zone_schd.Pop(n)

	// 如果该zone下无node, 则删除对应map
	if zone_schd.NumUnits() <= 0 {
		delete(self.zone_schds, zid)
	}
}

func (self *NasRoundRobinScheduler) DelNode(n *trans.Node) {
	// 删除失去连接的node
	if n.ZoneId == "" {
		return
	}
	self.mux.Lock()
	defer self.mux.Unlock()
	self.zoneDelNode(n.ZoneId, n)
}

func (self *NasRoundRobinScheduler) UpdateNode(n *trans.Node, req *transfer.KeepaliveReq) {
	// Do nothing
}

func (self *NasRoundRobinScheduler) NodeZoneChange(n *trans.Node, old_zid, new_zid string) {
	self.mux.Lock()
	defer self.mux.Unlock()
	if old_zid != "" {
		self.zoneDelNode(old_zid, n)
	}
	if new_zid != "" {
		self.zoneAddNode(new_zid, n)
	}
}

func (self *NasRoundRobinScheduler) dispatch(zid string) (n *trans.Node, err error) {
	self.mux.Lock()
	defer self.mux.Unlock()
	zone_schd := self.zone_schds[zid]
	if zone_schd == nil {
		err = fmt.Errorf("No such zone: %s", zid)
		return nil, err
	}
	u, err := zone_schd.Get(nil)
	if err != nil {
		return nil, err
	}
	n = u.(*trans.Node)
	return
}

func (self *NasRoundRobinScheduler) DispatchDownload(req *DownloadReq) (n *trans.Node, err error) {
	clog.Trace("in nas roundrobin dispatch download")
	url := req.url
	if url == nil {
		err = fmt.Errorf("no url")
		return nil, err
	}
	clog.Tracef("%#v", url)
	zid := url.Host
	return self.dispatch(zid)
}
