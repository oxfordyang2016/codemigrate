package models

import (
	"./../../utils/db"
	"cydex"
	"time"
)

// 区域, 多个Node在一个Zone,共享同一存储空间
type Zone struct {
	Id       int64     `xorm:"pk autoincr"`
	Zid      string    `xorm:"varchar(255) not null unique"`
	Name     string    `xorm:"varchar(255) not null"`
	Desc     string    `xorm:"text"`
	CreateAt time.Time `xorm:"DateTime created"`
	UpdateAt time.Time `xorm:"DateTime updated"`
	Nodes    []*Node   `xorm:"-"`
}

func CreateZone(zid, name, desc string) (*Zone, error) {
	z := &Zone{
		Zid:  zid,
		Name: name,
		Desc: desc,
	}
	if _, err := DB().Insert(z); err != nil {
		return nil, err
	}
	return z, nil
}

func CountZones() (int64, error) {
	n, err := DB().Count(new(Zone))
	return n, err
}

func GetZones(p *cydex.Pagination) ([]*Zone, error) {
	zones := make([]*Zone, 0)
	var err error
	sess := DB().NewSession()
	if p != nil {
		sess = sess.Limit(p.PageSize, (p.PageNum-1)*p.PageSize)
	}
	if err = sess.Find(&zones); err != nil {
		return nil, err
	}
	return zones, nil
}

func GetZone(zid string) (z *Zone, err error) {
	z = new(Zone)
	var existed bool
	if existed, err = DB().Where("zid=?", zid).Get(z); err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	return z, nil
}

// 删除zone
func DeleteZone(zoneid string) (err error) {
	z := &Zone{Zid: zoneid}
	has, err := DB().Get(z)
	if err != nil {
		return err
	}
	if !has {
		return nil
	}

	session := DB().NewSession()
	defer db.SessionRelease(session)
	if err := session.Begin(); err != nil {
		return err
	}
	// 相关node的ZoneId置为空
	sql := "update `transfer_node` set zone_id='' where zone_id=?"
	session.Exec(sql, zoneid)
	// delete self
	session.Id(z.Id).Delete(z)
	if err = session.Commit(); err != nil {
		return err
	}

	return nil
}

// name不能为空, 参数为空则不生效
func (self *Zone) SetName(name string) error {
	z := &Zone{
		Name: name,
	}
	_, err := DB().Id(self.Id).Update(z)
	return err
}

// desc可以为空
func (self *Zone) SetDesc(desc string) error {
	z := &Zone{
		Desc: desc,
	}
	_, err := DB().Id(self.Id).Cols("desc").Update(z)
	return err
}

func (self *Zone) AddNodes(nodes []string) error {
	if nodes == nil {
		return nil
	}
	n := new(Node)
	n.ZoneId = self.Zid
	_, err := DB().In("nid", nodes).Cols("zone_id").Update(n)
	return err
}

func (self *Zone) DelNodes(nodes []string) error {
	if nodes == nil {
		return nil
	}
	n := new(Node)
	_, err := DB().Where("zone_id=?", self.Zid).In("nid", nodes).Cols("zone_id").Update(n)
	return err
}

func (self *Zone) ClearNodes() error {
	n := new(Node)
	_, err := DB().Where("zone_id=?", self.Zid).Cols("zone_id").Update(n)
	return err
}

func (self *Zone) TableName() string {
	return "transfer_zone"
}
