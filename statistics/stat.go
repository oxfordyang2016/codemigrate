package statistics

var (
	TransferMgr *StatTransferManager
)

func init() {
	TransferMgr = NewStatTransferManager()
}
