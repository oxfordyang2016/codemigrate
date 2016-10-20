package pkg

import (
	// "./../transfer/task"
	// "./models"
	"cydex"
	// "cydex/transfer"
	"errors"
	"fmt"
	"sync"
	"time"
)

var lock sync.Mutex
var use_file_slice bool
var unpacker Unpacker

// 拆包器接口
type Unpacker interface {
	// 根据用户和请求生成Pid
	GeneratePid(uid string, title, notes string) (pid string)
	// 根据pid生成fid
	GenerateFid(pid string, index int, file *cydex.SimpleFile) (fid string, err error)
	// 根据file生成segs
	GenerateSegs(fid string, f *cydex.SimpleFile) (segs []*cydex.Seg, err error)
	// GetPidFromFid, 快速获取pid
	GetPidFromFid(fid string) string
	// Unpack可能会设置参数,这里要保护
	Enter()
	Leave()
}

// 默认拆包方法, 实现Unpacker接口
type DefaultUnpacker struct {
	// private
	min_seg_size   uint64 // in bytes
	max_seg_num    uint
	size_threshold uint64 // f.Size大于该值的, 按照最多max_seg_num拆分, 否则按照min_seg_size拆分
	start_no       int    // 起始序号
	lock           sync.Mutex
}

// min_seg_size: 最小分片size(bytes) max_seg_num: 最大分片数
func NewDefaultUnpacker(min_seg_size uint64, max_seg_num uint) *DefaultUnpacker {
	return &DefaultUnpacker{
		min_seg_size:   min_seg_size,
		max_seg_num:    max_seg_num,
		size_threshold: min_seg_size * uint64(max_seg_num),
		start_no:       1,
	}
}

func (self *DefaultUnpacker) GeneratePid(uid string, title, notes string) (pid string) {
	s := fmt.Sprintf("%d", time.Now().Unix())
	return uid + s
}

func (self *DefaultUnpacker) GenerateFid(pid string, index int, file *cydex.SimpleFile) (fid string, err error) {
	if index+self.start_no > 99 {
		return "", errors.New("too many files")
	}
	s := fmt.Sprintf("%02d", index+self.start_no)
	return pid + s, nil
}

func (self *DefaultUnpacker) GenerateSegs(fid string, f *cydex.SimpleFile) ([]*cydex.Seg, error) {
	if f.Type != cydex.FTYPE_FILE || f.Size == 0 {
		return nil, nil
	}

	var per_size uint64
	segs := make([]*cydex.Seg, 0, self.max_seg_num)
	if f.Size > self.size_threshold {
		if (f.Size % uint64(self.max_seg_num)) == 0 {
			per_size = f.Size / uint64(self.max_seg_num)
		} else {
			per_size = f.Size / uint64((self.max_seg_num - 1))
		}
	} else {
		per_size = self.min_seg_size
	}
	total_size := f.Size
	for idx := 0; total_size > 0; idx += 1 {
		size := per_size
		if total_size < per_size {
			size = total_size
		}
		seg := &cydex.Seg{
			Sid: fmt.Sprintf("%s%08d", fid, idx+self.start_no),
		}
		seg.SetSize(size)
		segs = append(segs, seg)
		total_size -= size
	}
	return segs, nil
}

func (self *DefaultUnpacker) GetPidFromFid(fid string) (pid string) {
	if len(fid) == 24 {
		pid = fid[:22]
	}
	return
}

func (self *DefaultUnpacker) Enter() {
	self.lock.Lock()
}

func (self *DefaultUnpacker) Leave() {
	self.lock.Unlock()
}

func (self *DefaultUnpacker) Config(min_seg_size uint64, max_seg_num uint) error {
	if min_seg_size == 0 || max_seg_num == 0 {
		return errors.New("Invalid seg num or seg size!")
	}
	self.Enter()
	defer self.Leave()
	self.min_seg_size = min_seg_size
	self.max_seg_num = max_seg_num
	self.size_threshold = min_seg_size * uint64(max_seg_num)
	return nil
}

func (self *DefaultUnpacker) GetConfig() (uint64, uint) {
	self.Enter()
	defer self.Leave()
	return self.min_seg_size, self.max_seg_num
}

func GetUnpacker() Unpacker {
	return unpacker
}

func SetUnpacker(u Unpacker) {
	if u != nil {
		unpacker = u
	}
}

func IsUsingFileSlice() bool {
	lock.Lock()
	defer lock.Unlock()
	return use_file_slice
}

func SetUsingFileSlice(enable bool) {
	lock.Lock()
	defer lock.Unlock()
	use_file_slice = enable
}
