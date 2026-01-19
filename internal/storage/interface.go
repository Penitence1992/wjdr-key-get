package storage

type KeyStorage interface {
	// IsReceived 检查code是否已经获取过
	IsReceived(fid, code string) (bool, error)
	// Save 保存获取记录
	Save(fid, code string) error
	// GetFids 获取用户id列表
	GetFids() ([]string, error)
	// SaveFidInfo 保存用户信息
	SaveFidInfo(fid int, nickname string, kid int, avatarImage string) error
	// AddTask 新增任务
	AddTask(code string) error
	// GetTask 获取未完成的任务
	GetTask() ([]string, error)
	// DoneTask 完成任务
	DoneTask(code string) error
}
