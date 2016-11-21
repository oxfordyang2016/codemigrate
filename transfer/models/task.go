package models

import (
	"time"
)

type Task struct {
	Id                       int64     `xorm:"pk autoincr"`
	TaskId                   string    `xorm:"varchar(255) not null unique"`
	Type                     int       `xorm:"not null"`
	JobId                    string    `xorm:"varchar(255) not null"`
	Pid                      string    `xorm:"varchar(255) not null"`
	Fid                      string    `xorm:"varchar(255) not null"`
	NodeId                   string    `xorm:"varchar(255) not null"`
	ZoneId                   string    `xorm:"varchar(255) not null"`
	NumSegs                  int       `xorm:"not null"`
	PeerIp                   string    `xorm:"varchar(32)"` // 对端的IP地址
	AllocatedBitrate         uint64    // 分配的码率
	AvgBitrate               uint64    // 平均码率
	MinBitrate               uint64    // 最小码率
	MaxBitrate               uint64    // 最大码率
	BitrateStandardDeviation float64   // 码率标准差
	State                    int       `xorm:"int default(0)"` // 状态
	CreateAt                 time.Time `xorm:"DateTime created"`
	UpdateAt                 time.Time `xorm:"DateTime updated"`
}

func CreateTask(t *Task) error {
	if t != nil {
		if _, err := DB().Insert(t); err != nil {
			return err
		}
	}
	return nil
}

func GetTask(task_id string) (t *Task, err error) {
	t = new(Task)
	var existed bool
	if existed, err = DB().Where("task_id=?", task_id).Get(t); err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	return t, nil
}

func GetTasksByJobId(jobid string) ([]*Task, error) {
	tasks := make([]*Task, 0)
	if err := DB().Where("job_id=?", jobid).Find(&tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (self *Task) UpdateState(state int) error {
	self.State = state
	_, err := DB().Id(self.Id).Cols("state").Update(self)
	return err
}

func (self *Task) UpdateAllocatedBitrate(v uint64) error {
	self.AllocatedBitrate = v
	_, err := DB().Id(self.Id).Cols("allocated_bitrate").Update(self)
	return err
}

func (self *Task) UpdateBitrateStatistics(min, avg, max uint64, standard_deviation float64) error {
	self.MinBitrate = min
	self.AvgBitrate = avg
	self.MaxBitrate = max
	self.BitrateStandardDeviation = standard_deviation
	_, err := DB().Id(self.Id).Cols("min_bitrate", "max_bitrate", "avg_bitrate", "bitrate_standard_deviation").Update(self)
	return err
}

func (self *Task) TableName() string {
	return "transfer_task"
}
