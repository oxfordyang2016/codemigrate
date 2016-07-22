package transfer

import (
	"./../db"
	"./models"
	"cydex"
	"cydex/transfer"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	// "runtime"
	// "fmt"
	"testing"
	"time"
)

const (
	TEST_DB = ":memory:"
	// TEST_DB = "/tmp/node_test.db"
)

// implement NodeObserver
type MyObserver struct {
	add bool
	del bool
}

func (o *MyObserver) AddNode(n *Node) {
	o.add = true
}
func (o *MyObserver) DelNode(n *Node) {
	o.del = true
}
func (o *MyObserver) UpdateNode(n *Node, req *transfer.KeepaliveReq) {
}

// func testNodeHandleRsp(node *Node, msg *transfer.Message) {
// 	_, _ := node.HandleMsg(msg)
// }

func Test_NodeManager(t *testing.T) {
	// runtime.GOMAXPROCS(4)

	Convey("Test Node Manager", t, func() {
		Convey("Test Add Node", func() {
			n := &Node{
				Node: &models.Node{
					Nid: "n1",
				},
				Token: "12345",
				Host:  "192.168.2.3",
			}
			NodeMgr.AddNode(n)
			n1 := NodeMgr.GetByNid("n1")
			So(n1, ShouldNotBeNil)
			So(n1.Nid, ShouldEqual, n.Nid)
			So(n1.Host, ShouldEqual, n.Host)
		})
		Convey("Test Del Node", func() {
			NodeMgr.DelNode("n1")
			n := NodeMgr.GetByNid("n1")
			So(n, ShouldBeNil)
		})

		Convey("Test observer", func() {
			var mo MyObserver
			NodeMgr.AddObserver(&mo)
			n := &Node{
				Node: &models.Node{
					Nid: "n1",
				},
				Token: "12345",
				Host:  "192.168.2.3",
			}
			NodeMgr.AddNode(n)
			So(mo.add, ShouldBeTrue)
			NodeMgr.DelNode("n1")
			So(mo.del, ShouldBeTrue)
		})
	})
}

func Test_Node(t *testing.T) {
	var err error
	var seq uint32
	var test_nid string
	var test_token string

	if TEST_DB != ":memory:" {
		os.Remove(TEST_DB)
	}

	Convey("Test Node", t, func() {
		Convey("DB Init using sqlite3", func() {
			Convey("create", func() {
				err = db.CreateDefaultDBEngine("sqlite3", TEST_DB, false)
				So(err, ShouldBeNil)
			})
			Convey("Sync Tables", func() {
				err = models.DBSyncTables()
				So(err, ShouldBeNil)
			})
		})

		Convey("Test Node register", func() {
			Convey("normal register", func() {
				seq++
				msg := transfer.NewReqMessage("", "register", "", seq)
				msg.Req = &transfer.Request{
					Register: &transfer.RegisterReq{
						MachineCode: "m1",
					},
				}
				node := NewNode(nil, nil)
				rsp, err := node.HandleMsg(msg)
				So(err, ShouldBeNil)
				So(rsp, ShouldNotBeNil)
				So(rsp.Seq, ShouldEqual, msg.Seq)
				So(rsp.Cmd, ShouldEqual, msg.Cmd)
				So(rsp.Rsp, ShouldNotBeNil)
				So(rsp.Rsp.Code, ShouldEqual, cydex.OK)
				So(rsp.Rsp.Register, ShouldNotBeNil)
				r := rsp.Rsp.Register
				So(r.Tnid, ShouldNotBeBlank)
				test_nid = r.Tnid
			})

			Convey("has registered", func() {
				seq++
				msg := transfer.NewReqMessage("", "register", "", seq)
				msg.Req = &transfer.Request{
					Register: &transfer.RegisterReq{
						MachineCode: "m1",
					},
				}
				node := NewNode(nil, nil)
				rsp, err := node.HandleMsg(msg)
				So(err, ShouldBeNil)
				So(rsp, ShouldNotBeNil)
				So(rsp.Seq, ShouldEqual, msg.Seq)
				So(rsp.Rsp, ShouldNotBeNil)
				So(rsp.Rsp.Code, ShouldEqual, cydex.OK)
				So(rsp.Rsp.Register, ShouldNotBeNil)
				r := rsp.Rsp.Register
				So(r.Tnid, ShouldEqual, test_nid)
			})
		})

		Convey("Test Node Login", func() {
			Convey("normal login", func() {
				seq++
				msg := transfer.NewReqMessage(test_nid, "login", "", seq)
				msg.Req = &transfer.Request{
					Login: &transfer.LoginReq{
						Version:      "1.0.0",
						OS:           "centos",
						TotalStorage: 1234,
						FreeStorage:  12,
					},
				}
				node := NewNode(nil, nil)
				rsp, err := node.HandleMsg(msg)
				So(err, ShouldBeNil)
				So(rsp, ShouldNotBeNil)
				So(rsp.Rsp, ShouldNotBeNil)
				So(rsp.Rsp.Code, ShouldEqual, cydex.OK)
				So(rsp.Rsp.Login, ShouldNotBeNil)
				login := rsp.Rsp.Login
				So(login.Token, ShouldNotBeBlank)
				test_token = login.Token
			})
			Convey("nid not existed", func() {
				seq++
				msg := transfer.NewReqMessage("nid_not_existed", "login", "", seq)
				msg.Req = &transfer.Request{
					Login: &transfer.LoginReq{
						Version:      "1.0.0",
						OS:           "centos",
						TotalStorage: 1234,
						FreeStorage:  12,
					},
				}
				node := NewNode(nil, nil)
				rsp, err := node.HandleMsg(msg)
				So(err, ShouldNotBeNil)
				So(rsp, ShouldNotBeNil)
				So(rsp.Rsp, ShouldNotBeNil)
				So(rsp.Rsp.Code, ShouldNotEqual, cydex.OK)
				So(rsp.Rsp.Login, ShouldBeNil)
			})
		})

		Convey("Test Node Keepalive", func() {
			Convey("normal keepalive", func() {
				seq++
				msg := transfer.NewReqMessage(test_nid, "keepalive", test_token, seq)
				msg.Req = &transfer.Request{
					Keepalive: &transfer.KeepaliveReq{
						CpuUsage:          89,
						TotalStorage:      1234,
						FreeStorage:       12,
						TotalMem:          128 * 1024 * 1024,
						FreeMem:           123 * 1024,
						UploadBandwidth:   100,
						DownloadBandwidth: 100,
					},
				}
				node := NodeMgr.GetByNid(test_nid)
				So(node, ShouldNotBeNil)
				rsp, err := node.HandleMsg(msg)
				So(err, ShouldBeNil)
				So(rsp, ShouldNotBeNil)
				So(rsp.Rsp, ShouldNotBeNil)
				So(rsp.Rsp.Code, ShouldEqual, cydex.OK)
			})
			Convey("invalid token", func() {
				seq++
				msg := transfer.NewReqMessage(test_nid, "keepalive", "invalid token", seq)
				msg.Req = &transfer.Request{
					Keepalive: &transfer.KeepaliveReq{
						CpuUsage:          89,
						TotalStorage:      1234,
						FreeStorage:       12,
						TotalMem:          128 * 1024 * 1024,
						FreeMem:           123 * 1024,
						UploadBandwidth:   100,
						DownloadBandwidth: 100,
					},
				}
				node := NodeMgr.GetByNid(test_nid)
				So(node, ShouldNotBeNil)
				rsp, err := node.HandleMsg(msg)
				So(err, ShouldNotBeNil)
				So(rsp, ShouldNotBeNil)
				So(rsp.Rsp, ShouldNotBeNil)
				So(rsp.Rsp.Code, ShouldNotEqual, cydex.OK)
			})
		})
		Convey("Test Sync Response", func() {
			Convey("normal", func() {
				node := NodeMgr.GetByNid(test_nid)
				msg := transfer.NewReqMessage(test_nid, "uploadtask", test_token, 0)
				msg.Req = &transfer.Request{
					UploadTask: &transfer.UploadTaskReq{
						TaskId:  "t1",
						Uid:     "u1",
						Fid:     "f1",
						SidList: []string{"s1", "s2", "s3"},
					},
				}

				go func(seq uint32) {
					m := transfer.NewRspMessage(test_nid, "uploadtask", test_token, seq)
					m.Rsp.Code = cydex.OK
					m.Rsp.UploadTask = &transfer.UploadTaskRsp{
						SidList:         []string{"s1", "s2", "s3"},
						SidStorage:      []string{"s1_s", "s2_s", "s3_s"},
						Port:            1234,
						RecomendBitrate: 123,
					}
					_, err := node.HandleMsg(m)
					err = err
				}(node.seq)

				rsp, err := node.SendRequestSync(msg, 5*time.Second)
				So(err, ShouldBeNil)
				So(rsp, ShouldNotBeNil)
				So(rsp.Rsp.UploadTask.Port, ShouldEqual, 1234)
			})
		})
		Convey("Test Node Close", func() {
			node := NodeMgr.GetByNid(test_nid)
			node.Close(true)
			n, err := models.GetNode(test_nid)
			So(n, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(time.Since(n.LastLogoutTime), ShouldBeLessThan, 1*time.Second)
		})
	})
}
