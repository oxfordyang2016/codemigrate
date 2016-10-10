package task

import (
	trans "./../"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

type FakeScheduleUnit struct {
	Id    int
	Value int
}

func NewFSU(id, v int) *FakeScheduleUnit {
	return &FakeScheduleUnit{
		Id:    id,
		Value: v,
	}
}

func filteById(u ScheduleUnit, args ...interface{}) bool {
	fsu := u.(*FakeScheduleUnit)
	id := args[0].(int)
	return fsu.Id == id
}

func filteNil(u ScheduleUnit, args ...interface{}) bool {
	if len(args) == 0 {
		return true
	}
	if args[0] == nil {
		return true
	}
	return false
}

func Test_RoundRobin(t *testing.T) {
	rr := NewRoundRobinScheduler()
	var fsus []*FakeScheduleUnit
	const num = 5

	Convey("Test RoundRobin", t, func() {
		Convey("put", func() {
			for i := 0; i < num; i++ {
				fsu := NewFSU(i+1, (i+1)*100)
				rr.Put(fsu)
				fsus = append(fsus, fsu)
			}
			So(rr.NumUnits(), ShouldEqual, num)
		})
		Convey("get", func() {
			u, err := rr.Get(filteById, 3)
			So(err, ShouldBeNil)
			So(u, ShouldNotBeNil)
			fsu := u.(*FakeScheduleUnit)
			So(fsu.Id, ShouldEqual, 3)

			u, err = rr.Get(filteNil, nil)
			So(err, ShouldBeNil)
			So(u, ShouldNotBeNil)
			fsu = u.(*FakeScheduleUnit)
			So(fsu.Id, ShouldEqual, 1)
		})
		Convey("pop", func() {
			rr.Pop(fsus[1])
			So(rr.NumUnits(), ShouldEqual, num-1)

			rr.Pop(NewFSU(2, 5))
			So(rr.NumUnits(), ShouldEqual, num-1)
		})
	})
}
