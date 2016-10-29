package models

import (
	"./../../utils/db"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
)

const (
	TEST_DB = ":memory:"
	// TEST_DB = "/tmp/zone_test.db"
)

func init() {
	if TEST_DB != ":memory:" {
		os.Remove(TEST_DB)
	}
	db.CreateEngine("sqlite3", TEST_DB, false)
	SyncTables()
}

func Test_Zone(t *testing.T) {
	Convey("Test zone", t, func() {
		Convey("create", func() {
			zone, err := CreateZone("z1", "z1", "")
			So(err, ShouldBeNil)
			So(zone, ShouldNotBeNil)
			So(zone.Zid, ShouldEqual, "z1")
			So(zone.Name, ShouldEqual, "z1")
			So(zone.Desc, ShouldBeEmpty)
		})
		Convey("get", func() {
			zone, err := GetZone("z1")
			So(err, ShouldBeNil)
			So(zone, ShouldNotBeNil)
		})
		Convey("modify name", func() {
			zone, _ := GetZone("z1")
			zone.SetName("z1_name")

			zone, _ = GetZone("z1")
			So(zone.Name, ShouldEqual, "z1_name")
			zone.SetName("")

			zone, _ = GetZone("z1")
			So(zone.Name, ShouldEqual, "z1_name")
		})
		Convey("modify desc", func() {
			zone, _ := GetZone("z1")
			zone.SetDesc("desc")

			zone, _ = GetZone("z1")
			So(zone.Desc, ShouldEqual, "desc")
			zone.SetDesc("")

			zone, _ = GetZone("z1")
			So(zone.Desc, ShouldBeEmpty)
		})
		Convey("delete", func() {
			const node_count = 5
			for i := 0; i < node_count; i++ {
				mc := fmt.Sprintf("mc_%d", i+1)
				node_id := fmt.Sprintf("n%d", i+1)
				n, err := CreateNode(mc, node_id)
				So(err, ShouldBeNil)
				So(n, ShouldNotBeNil)
				err = n.SetZone("z1")
				So(err, ShouldBeNil)

				n, _ = GetNode("n1")
				So(n.ZoneId, ShouldEqual, "z1")
			}

			err := DeleteZone("z1")
			So(err, ShouldBeNil)

			for i := 0; i < node_count; i++ {
				node_id := fmt.Sprintf("n%d", i+1)
				n, _ := GetNode(node_id)
				So(n.ZoneId, ShouldBeEmpty)
			}
		})
	})
}

func Test_ZoneNodes(t *testing.T) {
	Convey("Test zone node relationship", t, func() {
		Convey("add and del", func() {
			const zone_count = 3
			for i := 0; i < zone_count; i++ {
				zid := fmt.Sprintf("z_%d", i+1)
				name := fmt.Sprintf("z%d name", i+1)
				_, err := CreateZone(zid, name, "")
				So(err, ShouldBeNil)
			}

			const node_count = 5
			for i := 0; i < node_count; i++ {
				mc := fmt.Sprintf("mc222_%d", i+1)
				node_id := fmt.Sprintf("n_%d", i+1)
				n, err := CreateNode(mc, node_id)
				So(err, ShouldBeNil)
				So(n, ShouldNotBeNil)
			}

			// z_1 add [n_1, n_2]
			z1, _ := GetZone("z_1")
			So(z1, ShouldNotBeNil)
			err := z1.AddNodes([]string{"n_1", "n_2"})
			So(err, ShouldBeNil)
			for _, nid := range []string{"n_1", "n_2"} {
				n, _ := GetNode(nid)
				So(n.ZoneId, ShouldEqual, "z_1")
			}

			// z_2 add [n_3]
			z2, _ := GetZone("z_2")
			So(z2, ShouldNotBeNil)
			z2.AddNodes([]string{"n_3"})

			// z_1 del n_2, Note n_3 is belong z_2
			err = z1.DelNodes([]string{"n_2", "n_3"})
			So(err, ShouldBeNil)
			for _, nid := range []string{"n_1", "n_2", "n_3"} {
				n, _ := GetNode(nid)
				switch nid {
				case "n_1":
					So(n.ZoneId, ShouldEqual, "z_1")
				case "n_2":
					So(n.ZoneId, ShouldBeEmpty)
				case "n_3":
					So(n.ZoneId, ShouldEqual, "z_2")
				}
			}
		})
		Convey("clear", func() {
			z1, _ := GetZone("z_1")
			err := z1.ClearNodes()
			So(err, ShouldBeNil)
			for _, nid := range []string{"n_1", "n_2", "n_3"} {
				n, _ := GetNode(nid)
				switch nid {
				case "n_1":
					So(n.ZoneId, ShouldBeEmpty)
				case "n_2":
					So(n.ZoneId, ShouldBeEmpty)
				case "n_3":
					So(n.ZoneId, ShouldEqual, "z_2")
				}
			}
		})
	})
}
