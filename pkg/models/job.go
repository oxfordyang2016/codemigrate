package models

import (
	"cydex"
	"errors"
	"fmt"
	"time"
)

const (
	SOFT_DELETE_TAG           = 1 //标记
	SOFT_DELETE_FILES_REMOVED = 2 //TN上文件已删除
)

type Job struct {
	Id            int64     `xorm:"pk autoincr"`
	JobId         string    `xorm:"unique not null"`
	Type          int       `xorm:"int"` // cydex.UPLOAD or cydex.DOWNLOAD
	Pid           string    `xorm:"varchar(22) not null"`
	Pkg           *Pkg      `xorm:"-"`
	Uid           string    `xorm:"varchar(12) not null"`
	CreateAt      time.Time `xorm:"DateTime created"`
	UpdateAt      time.Time `xorm:"DateTime updated"`
	SoftDel       int       `xorm:"BOOL not null default(0)"`
	FinishedTimes int       `xorm:"not null default(0)"`
	State         int       `xorm:"Int not null default(0)"` //FIXME: 目前和jd状态不同步，只用于判断是否是第一次下载

	// runtime usage
	Details              map[string]*JobDetail `xorm:"-"`
	NumUnfinishedDetails int64                 `xorm:"-"`
	IsCached             bool                  `xorm:"-"`
}

type JobFilter struct {
	Owner   string
	BegTime time.Time
	EndTime time.Time
	Title   string
	OrderBy string // 排序字符串
}

func CreateJob(jobid, uid, pid string, typ int) (*Job, error) {
	j := &Job{
		JobId: jobid,
		Uid:   uid,
		Pid:   pid,
		Type:  typ,
	}
	if _, err := DB().Insert(j); err != nil {
		return nil, err
	}
	return j, nil
}

func GetJob(jobid string, with_pkg bool) (*Job, error) {
	j := new(Job)
	existed, err := DB().Where("job_id=?", jobid).Get(j)
	if err != nil || !existed {
		return nil, nil
	}
	if with_pkg {
		err = j.GetPkg(true)
	}
	return j, err
}

func GetJobs(typ int, p *cydex.Pagination) ([]*Job, error) {
	fmt.Println(`
              UUUUUUUUUUUUU
                 VVVVVVV
                  VVVVV
                    V
func GetJobs(typ int, p *cydex.Pagination) ([]*Job, error) {
	              UUUUU
	              UUUUU
	                VVVVV
	                VVV
		`)
	jobs := make([]*Job, 0)
	var err error
	query := "type=? and soft_del=0"
	if p != nil {
		n, _ := DB().Where(query, typ).Count(new(Job))
		p.TotalNum = n
	}
	sess := DB().Where(query, typ)
	if p != nil {
		sess = sess.Limit(p.PageSize, (p.PageNum-1)*p.PageSize)
	}
	sess = sess.Desc("create_at")
	if err = sess.Find(&jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func GetJobsEx(typ int, p *cydex.Pagination, filter *JobFilter) ([]*Job, error) {
	/*
type Job struct {
	Id            int64     `xorm:"pk autoincr"`
	JobId         string    `xorm:"unique not null"`
	Type          int       `xorm:"int"` // cydex.UPLOAD or cydex.DOWNLOAD
	Pid           string    `xorm:"varchar(22) not null"`
	Pkg           *Pkg      `xorm:"-"`
	Uid           string    `xorm:"varchar(12) not null"`
	CreateAt      time.Time `xorm:"DateTime created"`
	UpdateAt      time.Time `xorm:"DateTime updated"`
	SoftDel       int       `xorm:"BOOL not null default(0)"`
	FinishedTimes int       `xorm:"not null default(0)"`
	State         int       `xorm:"Int not null default(0)"` //FIXME: 目前和jd状态不同步，只用于判断是否是第一次下载

	// runtime usage
	Details              map[string]*JobDetail `xorm:"-"`
	NumUnfinishedDetails int64                 `xorm:"-"`
	IsCached             bool                  `xorm:"-"`
}

*/
	fmt.Println(`
                       |
                       |
                       |

		---i have enter func GetJobsEx(typ int,

	    p *cydex.Pagination, filter *JobFilter)
	  
	             ([]*Job, error) { 
                       |
                       |
                       v


	             	`)
	jobs := make([]*Job, 0)

	has_orderby := false
/*
func DB() *xorm.Engine {
	return db.Engine
}

*/
	sess := DB().NewSession()//this is db initiation
	sess = sess.Where("package_job.type=? and package_job.soft_del=0", typ)
    fmt.Println(`
    	         ||||
              UUUUUUUUUUUUU
               yyyyyyyyy
                  VVV
                  VVV

             i wlll in there query database 
                |||||
                  ||
                  VVV    
    	`)
	if filter != nil {
		//this is sql 
		sess = sess.Join("INNER", "package_pkg", "package_pkg.pid = package_job.pid")
		if filter.Title != "" {
			// sess = sess.Where("package_pkg.title like ?", fmt.Sprintf("'%%%s%%'", filter.Title))
			// FIXME 这里应该使用占位符更安全
			sess = sess.Where(fmt.Sprintf("package_pkg.title like '%%%s%%'", filter.Title))
		}
		if !filter.BegTime.IsZero() || !filter.EndTime.IsZero() {
			var beg time.Time
			end := time.Now()
			if !filter.BegTime.IsZero() {
				beg = filter.BegTime
			}
			if !filter.EndTime.IsZero() {
				end = filter.EndTime
			}
			sess = sess.Where("package_pkg.create_at >= ? and package_pkg.create_at <= ?", beg, end)
		}
		if filter.Owner != "" && typ == cydex.UPLOAD {
			sess = sess.Where("package_job.uid = ?", filter.Owner)
		}

   


		if filter.OrderBy != "" {
			has_orderby = true
			order := "ASC"
			order_item := filter.OrderBy
			if filter.OrderBy[0] == '-' {
				order = "DESC"
				order_item = filter.OrderBy[1:]
			}
			if order_item == "create" {
				order_item = "create_at"
			}
			order_by := fmt.Sprintf("package_pkg.%s %s", order_item, order)
			sess = sess.OrderBy(order_by)
		}
	}

	if !has_orderby {
		sess = sess.Desc("package_job.create_at")
	}
	if p != nil {
		n, _ := sess.Count(new(Job))
		p.TotalNum = n
	}


	//this is quering jobs.
	if err := sess.Find(&jobs); err != nil {
		return nil, err
	}
     

    if p != nil {
//`		sess = sess.Limit(p.PageSize, (p.PageNum-1)*p.PageSize)
    	sess = sess.Limit(1,1)

	}  

	return jobs, nil
}

func GetJobsByUid(uid string, typ int, p *cydex.Pagination) ([]*Job, error) {
	jobs := make([]*Job, 0)
	var err error
	query := "uid=? and type=? and soft_del=0"
	if p != nil {
		n, _ := DB().Where(query, uid, typ).Count(new(Job))
		p.TotalNum = n
	}
	sess := DB().Where(query, uid, typ)
	if p != nil {
		sess = sess.Limit(p.PageSize, (p.PageNum-1)*p.PageSize)
	}
	sess = sess.Desc("create_at")
	if err = sess.Find(&jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func GetJobsByPid(pid string, typ int, p *cydex.Pagination) ([]*Job, error) {
	jobs := make([]*Job, 0)
	var err error
	query := "pid=? and type=? and soft_del=0"
	if p != nil {
		n, _ := DB().Where(query, pid, typ).Count(new(Job))
		p.TotalNum = n
	}
	sess := DB().Where(query, pid, typ)
	if p != nil {
		sess = sess.Limit(p.PageSize, (p.PageNum-1)*p.PageSize)
	}
	if err = sess.Find(&jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

// 查询未完成的任务
func GetUnFinishedJobs() (ret []*Job, err error) {
	jobs := make([]*Job, 0)
	if err = DB().Where("soft_del=0").Find(&jobs); err != nil {
		return nil, err
	}
	for _, j := range jobs {
		n, err := CountUnfinishedJobDetails(j.JobId)
		if err != nil {
			continue
		}
		if n > 0 {
			ret = append(ret, j)
		}
	}
	return ret, nil
}

// 删除job
func DeleteJob(jobid string) (err error) {
	j := &Job{JobId: jobid}
	has, err := DB().Get(j)
	if err != nil {
		return err
	}
	if !has {
		return nil
	}

	session := DB().NewSession()
	defer SessionRelease(session)
	if err := session.Begin(); err != nil {
		return err
	}
	jds := make([]*JobDetail, 0, 100)
	if err = session.Where("job_id=?", jobid).Find(&jds); err != nil {
		return err
	}
	for _, jd := range jds {
		if _, err = session.Id(jd.Id).Delete(jd); err != nil {
			return err
		}
	}
	session.Id(j.Id).Delete(j)
	if err = session.Commit(); err != nil {
		return err
	}

	return nil
}

func (self *Job) TableName() string {
	return "package_job"
}

func (self *Job) GetDetails() error {
	jds, err := GetJobDetails(self.JobId)
	if err != nil {
		return err
	}
	if self.Details == nil {
		self.Details = make(map[string]*JobDetail)
	}
	for _, jd := range jds {
		self.Details[jd.Fid] = jd
	}
	return err
}

func (self *Job) CountUnfinishedDetails() int64 {
	n, _ := CountUnfinishedJobDetails(self.JobId)
	return n
}

func (self *Job) GetDetail(fid string) *JobDetail {
	if self.Details == nil {
		self.Details = make(map[string]*JobDetail)
	}
	jd, err := GetJobDetail(self.JobId, fid)
	if err != nil {
		return nil
	}
	return jd
}

func (self *Job) GetPkg(with_files bool) (err error) {
	if self.Pkg, err = GetPkg(self.Pid, with_files); err != nil {
		return
	}
	return
}

// 软删除
func (self *Job) SoftDelete(tag int) (err error) {
	if tag != SOFT_DELETE_TAG && tag != SOFT_DELETE_FILES_REMOVED {
		return fmt.Errorf("unsupport soft delete tag:%d", tag)
	}
	j := &Job{
		SoftDel: tag,
	}
	_, err = DB().Where("job_id=?", self.JobId).Cols("soft_del").Update(j)
	return err
}

func (self *Job) IsFinished() bool {
	n, _ := CountUnfinishedJobDetails(self.JobId)
	return n == 0
}

func (self *Job) Finish() error {
	self.FinishedTimes++
	self.State = cydex.TRANSFER_STATE_DONE
	_, err := DB().Id(self.Id).Cols("finished_times", "state").Update(self)
	return err
}

func (self *Job) SaveState(state int) error {
	self.State = state
	_, err := DB().Id(self.Id).Cols("state").Update(self)
	return err
}

// 获取传输了多少数据
func (self *Job) GetTransferedSize() (uint64, error) {
	jds, err := GetJobDetails(self.JobId)
	if err != nil {
		return 0, err
	}
	total := uint64(0)
	for _, jd := range jds {
		if jd.State == cydex.TRANSFER_STATE_DONE {
			total += jd.FinishedSize
		}
	}
	return total, nil
}

func (self *Job) String() string {
	return fmt.Sprintf("<Job(%s)>", self.JobId)
}

// ----------------------------------------------------------------------------- //

type JobDetail struct {
	Id              int64     `xorm:"pk autoincr"`
	JobId           string    `xorm:"unique(jobidfid) not null"`
	Job             *Job      `xorm:"-"`
	Fid             string    `xorm:"unique(jobidfid) varchar(24) not null "`
	File            *File     `xorm:"-"`
	StartTime       time.Time `xorm:"DateTime"`
	FinishTime      time.Time `xorm:"DateTime"`
	FinishedSize    uint64    `xorm:"BigInt not null default(0)"`
	State           int       `xorm:"Int not null default(0)"`
	NumFinishedSegs int       `xorm:"not null default(0)"`
	Checked         int       `xorm:"Int not null default(0)"`
	CreateAt        time.Time `xorm:"DateTime created"`
	UpdateAt        time.Time `xorm:"DateTime updated"`

	// runtime
	Bitrate     uint64          `xorm:"-"`
	CurSegSize  uint64          `xorm:"-"`
	Segs        map[string]*Seg `xorm:"-"` //sid->seg
	Err307Times int             `xorm:"-"`
}

// 批量创建
func CreateJobDetails(jds []*JobDetail) error {
	if jds == nil {
		return errors.New("job details slice is nil")
	}
	_, err := DB().Insert(&jds)
	return err
}

func CreateJobDetail(job_id, fid string) (*JobDetail, error) {
	d := &JobDetail{
		JobId: job_id,
		Fid:   fid,
	}
	if _, err := DB().Insert(d); err != nil {
		return nil, err
	}
	return d, nil
}

func GetJobDetail(jobid, fid string) (jd *JobDetail, err error) {
	jd = new(JobDetail)
	var existed bool
	if existed, err = DB().Where("job_id=? and fid=?", jobid, fid).Get(jd); err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil
	}
	return jd, nil
}

func GetJobDetails(job_id string) ([]*JobDetail, error) {
	ds := make([]*JobDetail, 0)
	if err := DB().Where("job_id=?", job_id).Find(&ds); err != nil {
		return nil, err
	}
	return ds, nil
}

func CountUnfinishedJobDetails(job_id string) (int64, error) {
	jd := new(JobDetail)
	n, err := DB().Where("job_id=? and state!=?", job_id, cydex.TRANSFER_STATE_DONE).Count(jd)
	return n, err
}

func GetJobDetailsByFid(fid string) ([]*JobDetail, error) {
	ds := make([]*JobDetail, 0)
	if err := DB().Where("fid=?", fid).Find(&ds); err != nil {
		return nil, err
	}
	return ds, nil
}

// NOTE: 启动时所有DOING的state均置为PAUSE。如果留有DOING的状态，那么会造成无法再传输的问题
func ProtectJobDetailState() error {
	jd := new(JobDetail)
	jd.State = cydex.TRANSFER_STATE_PAUSE
	_, err := DB().Where("state=?", cydex.TRANSFER_STATE_DOING).Cols("state").Update(jd)
	return err
}

func (self *JobDetail) GetFile() (err error) {
	var f *File
	if f, err = GetFile(self.Fid); err != nil {
		return err
	}
	self.File = f
	return
}

func (self *JobDetail) SetStartTime(t time.Time) error {
	if t.IsZero() {
		t = time.Now()
	}
	jd := &JobDetail{
		StartTime: t,
	}
	_, err := DB().Id(self.Id).Update(jd)
	if err == nil {
		self.StartTime = t
	}
	return err
}

func (self *JobDetail) SetState(state int) error {
	jd := &JobDetail{
		State: state,
	}
	_, err := DB().Id(self.Id).Cols("state").Update(jd)
	if err == nil {
		self.State = state
	}
	return err
}

func (self *JobDetail) Finish() error {
	jd := &JobDetail{
		State:      cydex.TRANSFER_STATE_DONE,
		FinishTime: time.Now(),
	}
	_, err := DB().Id(self.Id).Cols("state", "finish_time").Update(jd)
	if err == nil {
		self.State = jd.State
		self.FinishTime = jd.FinishTime
	}
	return err
}

// 全保存
func (self *JobDetail) Save() error {
	_, err := DB().Id(self.Id).AllCols().Update(self)
	return err
}

// 因为任务完成后可以重新开始
func (self *JobDetail) Reset() error {
	self.State = cydex.TRANSFER_STATE_IDLE
	self.StartTime = time.Time{}
	self.FinishTime = time.Time{}
	self.FinishedSize = 0
	self.NumFinishedSegs = 0
	self.Checked = 0
	return self.Save()
}

func (self *JobDetail) TableName() string {
	return "package_job_detail"
}

func (self *JobDetail) String() string {
	return fmt.Sprintf("<JobDetail(%s:%s)", self.JobId, self.Fid)
}
