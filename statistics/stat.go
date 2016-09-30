package statistics

var (
	TransferMgr *StatTransferManager
	PkgMgr      *StatPkgManager
)

func init() {
	TransferMgr = NewStatTransferManager()
	PkgMgr = new(StatPkgManager)
}
