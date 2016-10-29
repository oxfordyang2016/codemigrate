package task

import (
	trans "./../"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func createNode(nid, zid string) *trans.Node {
	node := trans.NewNode(nil, nil)
	node.Nid = nid
	node.ZoneId = zid
	return node
}

func Test_NasRR(t *testing.T) {
	Convey("Test nas roundrobin scheduler", t, func() {
		Convey("zone with one node schedule", func() {
			nas := NewNasRoundRobinScheduler()
			node_mgr := trans.NewNodeManager()
			node_mgr.AddObserver(nas)

			// z1: [n1, n2, n3]
			nodes := []*trans.Node{
				createNode("n1", "z1"),
			}
			for _, n := range nodes[:] {
				node_mgr.AddNode(n)
			}

			So(nas.zone_schds["z1"].NumUnits(), ShouldEqual, 1)
			for i := 0; i < 10; i++ {
				n, err := nas.dispatch("z1")
				So(err, ShouldBeNil)
				So(n, ShouldEqual, nodes[0])
			}
		})
		Convey("zone with multi nodes schedule", func() {
			nas := NewNasRoundRobinScheduler()
			node_mgr := trans.NewNodeManager()
			node_mgr.AddObserver(nas)

			// z1: [n1, n2, n3]
			nodes := [3]*trans.Node{
				createNode("n1", "z1"),
				createNode("n2", "z1"),
				createNode("n3", "z1"),
			}
			for _, n := range nodes[:] {
				node_mgr.AddNode(n)
			}

			So(nas.zone_schds["z1"].NumUnits(), ShouldEqual, 3)
			for i := 0; i < 10; i++ {
				n, err := nas.dispatch("z1")
				So(err, ShouldBeNil)
				So(n, ShouldEqual, nodes[i%3])
			}
		})
		Convey("node deleted", func() {
			nas := NewNasRoundRobinScheduler()
			node_mgr := trans.NewNodeManager()
			node_mgr.AddObserver(nas)

			// z1: [n1, n2, n3]
			nodes := [3]*trans.Node{
				createNode("n1", "z1"),
				createNode("n2", "z1"),
				createNode("n3", "z1"),
			}
			for _, n := range nodes[:] {
				node_mgr.AddNode(n)
			}

			So(nas.zone_schds["z1"].NumUnits(), ShouldEqual, 3)
			// jzh: 删除了"n2"
			node_mgr.DelNode("n2")
			So(nas.zone_schds["z1"].NumUnits(), ShouldEqual, 2)

			for i := 0; i < 10; i++ {
				n, err := nas.dispatch("z1")
				So(err, ShouldBeNil)
				if i%2 == 0 {
					So(n, ShouldEqual, nodes[0])
				} else {
					So(n, ShouldEqual, nodes[2])
				}
			}
		})
		Convey("zone map should clear when no more nodes", func() {
			nas := NewNasRoundRobinScheduler()
			node_mgr := trans.NewNodeManager()
			node_mgr.AddObserver(nas)

			nodes := [1]*trans.Node{
				createNode("n1", "z1"),
			}
			for _, n := range nodes[:] {
				node_mgr.AddNode(n)
			}

			So(nas.zone_schds, ShouldHaveLength, 1)
			So(nas.zone_schds["z1"].NumUnits(), ShouldEqual, 1)
			node_mgr.DelNode("n2")
			So(nas.zone_schds["z1"].NumUnits(), ShouldEqual, 1)
			node_mgr.DelNode("n1")
			_, ok := nas.zone_schds["z1"]
			So(ok, ShouldBeFalse)
			So(nas.zone_schds, ShouldHaveLength, 0)
		})
		Convey("node zone change", func() {
			nas := NewNasRoundRobinScheduler()
			node_mgr := trans.NewNodeManager()
			node_mgr.AddObserver(nas)

			// z1: [n1, n2, n3]
			nodes := []*trans.Node{
				createNode("n1", "z1"),
				createNode("n2", "z1"),
				createNode("n3", "z2"),
				createNode("n4", ""),
			}
			for _, n := range nodes[:] {
				node_mgr.AddNode(n)
			}

			So(nas.zone_schds, ShouldHaveLength, 2)
			So(nas.zone_schds["z1"].NumUnits(), ShouldEqual, 2)
			So(nas.zone_schds["z2"].NumUnits(), ShouldEqual, 1)
			node_mgr.NodeZoneChange(nodes[3], "", "z2")
			So(nas.zone_schds["z2"].NumUnits(), ShouldEqual, 2)
			So(nas.zone_schds["z2"].NumUnits(), ShouldEqual, 2)

			for i := 0; i < 10; i++ {
				n, err := nas.dispatch("z2")
				So(err, ShouldBeNil)
				if i%2 == 0 {
					So(n, ShouldEqual, nodes[2])
				} else {
					So(n, ShouldEqual, nodes[3])
				}
			}
		})
	})
}
