package statistics

import (
	"./../pkg"
	pkg_model "./../pkg/models"
	"cydex"
	clog "github.com/cihub/seelog"
	"strconv"
)

type StatPkgManager struct {
}

func (self *StatPkgManager) Get() (ret *cydex.PkgStat) {
	ret = new(cydex.PkgStat)

	sql := "select count(*), sum(num_files), sum(size) from package_pkg join package_job on package_job.Pid=package_pkg.Pid where package_job.soft_del=0 and package_job.Type=1"
	results, err := pkg_model.DB().Query(sql)
	if err != nil {
		clog.Error(err)
	}
	if len(results) > 0 {
		m := results[0]
		var v int
		v, _ = strconv.Atoi(string(m["count(*)"]))
		ret.TotalCnt = uint64(v)
		v, _ = strconv.Atoi(string(m["sum(num_files)"]))
		ret.TotalFiles = uint64(v)
		v, _ = strconv.Atoi(string(m["sum(size)"]))
		ret.TotalSize = uint64(v)
	}

	sql = "select count(*) from package_job where soft_del!=0 and type=1"
	results, err = pkg_model.DB().Query(sql)
	if err != nil {
		clog.Error(err)
	}
	if len(results) > 0 {
		m := results[0]
		v, _ := strconv.Atoi(string(m["count(*)"]))
		ret.TotalDelCnt = uint64(v)
	}

	// NOTE: 取未完成的jobs,上传查询数据库,下载使用track里的.
	// 因为track里没下载完,对应的上传还会留在track里,不能用在此处
	sql = "select count(*) from package_job join package_job_detail on package_job_detail.job_id=package_job.job_id where package_job.soft_del=0 and package_job.type=1 and package_job_detail.state!=2"
	results, err = pkg_model.DB().Query(sql)
	if err != nil {
		clog.Error(err)
	}
	if len(results) > 0 {
		m := results[0]
		v, _ := strconv.Atoi(string(m["count(*)"]))
		ret.UnfinishedUploadJobs = uint(v)
	}
	_, d := pkg.JobMgr.GetUnFinishedJobCount()
	ret.UnfinishedDownloadJobs = uint(d)

	return
}
