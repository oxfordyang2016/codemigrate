package models

import (
	"cydex"
	"time"
)

type Node struct {
	Id             int64  `xorm:"pk autoincr"`
	MachineCode    string `xorm:"varchar(255) not null unique"`
	Nid            string `xorm:"varchar(64) not null unique"`
	Name           string `xorm:"varchar(255)"`
	ZoneId         string `xorm:"varchar(255)"`
	PublicAddr     string `xorm:"varchar(64)"`
	RxBandwidth    uint64
	TxBandwidth    uint64
	RegisterTime   time.Time `xorm:"-"`
	LastLoginTime  time.Time `xorm:"DateTime"`
	LastLogoutTime time.Time `xorm:"DateTime"`
	CreateAt       time.Time `xorm:"DateTime created"`
	UpdateAt       time.Time `xorm:"DateTime updated"`
	Zone           *Zone     `xorm:"-"`
}

func CreateNode(machine_code, nid string) (*Node, error) {
	n := &Node{
		MachineCode: machine_code,
		Nid:         nid,
	}
	if _, err := DB().Insert(n); err != nil {
		return nil, err
	}
	return n, nil
}

func GetNode(nid string) (n *Node, err error) {
	n = new(Node)
	var existed bool
	if existed, err = DB().Where("nid=?", nid).Get(n); err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	n.RegisterTime = n.CreateAt
	return n, nil
}

func GetNodeByMachineCode(mc string) (n *Node, err error) {
	n = new(Node)
	var existed bool
	if existed, err = DB().Where("machine_code=?", mc).Get(n); err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	n.RegisterTime = n.CreateAt
	return n, nil
}

func CountNodes() (int64, error) {
	n, err := DB().Count(new(Node))
	return n, err
}

func GetNodes(p *cydex.Pagination) ([]*Node, error) {
	nodes := make([]*Node, 0)
	var err error
	sess := DB().NewSession()
	if p != nil {
		sess = sess.Limit(p.PageSize, (p.PageNum-1)*p.PageSize)
	}
	if err = sess.Find(&nodes); err != nil {
		return nil, err
	}
	for _, n := range nodes {
		n.RegisterTime = n.CreateAt
	}
	return nodes, nil
}

func GetNidsByZone(zid string) ([]string, error) {
	node := new(Node)
	rows, err := DB().Where("zone_id=?", zid).Rows(node)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nid_list []string
	for rows.Next() {
		if err = rows.Scan(node); err != nil {
			return nil, err
		}
		nid_list = append(nid_list, node.Nid)
	}
	return nid_list, nil
}

func GetNodesByZone(zid string) ([]*Node, error) {
	nodes := make([]*Node, 0)
	if err := DB().Where("zone_id=?", zid).Find(&nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (n *Node) SetName(name string) error {
	n.Name = name
	_, err := DB().Id(n.Id).Cols("name").Update(n)
	return err
}

func (n *Node) SetZone(zone_id string) error {
	n.ZoneId = zone_id
	_, err := DB().Id(n.Id).Cols("zone_id").Update(n)
	return err
}

func (n *Node) SetPublicAddr(v string) error {
	n.PublicAddr = v
	_, err := DB().Id(n.Id).Cols("public_addr").Update(n)
	return err
}

func (n *Node) SetRxBandwidth(v uint64) error {
	n.RxBandwidth = v
	_, err := DB().Id(n.Id).Cols("rx_bandwidth").Update(n)
	return err
}

func (n *Node) SetTxBandwidth(v uint64) error {
	n.TxBandwidth = v
	_, err := DB().Id(n.Id).Cols("tx_bandwidth").Update(n)
	return err
}

func (n *Node) UpdateLoginTime(t time.Time) error {
	n.LastLoginTime = t
	if _, err := DB().Where("nid=?", n.Nid).Update(n); err != nil {
		return err
	}
	return nil
}

func (n *Node) UpdateLogoutTime(t time.Time) error {
	n.LastLogoutTime = t
	if _, err := DB().Where("nid=?", n.Nid).Update(n); err != nil {
		return err
	}
	return nil
}

func (self *Node) GetZone() error {
	var err error
	if self.ZoneId != "" {
		self.Zone, err = GetZone(self.ZoneId)
	}
	return err
}

func (self *Node) TableName() string {
	return "transfer_node"
}
