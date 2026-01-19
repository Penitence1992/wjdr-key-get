package job

import (
	"cdk-get/internal/svc"

	"github.com/sirupsen/logrus"
)

var globalScheduler *Scheduler

// InitTask 初始化任务调度
func InitTask(svcCtx *svc.ServiceContext) error {
	globalScheduler = NewScheduler()

	// 添加任务
	globalScheduler.AddJob(NewGetCodeJob(svcCtx))

	// 启动调度器
	if err := globalScheduler.Start(); err != nil {
		logrus.WithError(err).Error("failed to start scheduler")
		return err
	}

	logrus.Info("task scheduler initialized successfully")
	return nil
}

// StopTask 停止任务调度
func StopTask() error {
	if globalScheduler != nil {
		return globalScheduler.Stop()
	}
	return nil
}
