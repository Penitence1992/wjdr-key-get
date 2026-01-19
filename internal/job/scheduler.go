package job

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Job 任务接口
type Job interface {
	Run(ctx context.Context)
	DelayTime() time.Duration
	PeriodTime() time.Duration
	Name() string
}

// Scheduler 任务调度器
type Scheduler struct {
	jobs   []Job
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// NewScheduler 创建新的调度器
func NewScheduler() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		jobs:   make([]Job, 0),
		ctx:    ctx,
		cancel: cancel,
	}
}

// AddJob 添加任务到调度器
func (s *Scheduler) AddJob(job Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	s.mu.Lock()
	jobs := make([]Job, len(s.jobs))
	copy(jobs, s.jobs)
	s.mu.Unlock()

	for _, job := range jobs {
		s.wg.Add(1)
		go s.runJob(job)
	}

	logrus.Infof("scheduler started with %d jobs", len(jobs))
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() error {
	logrus.Info("stopping scheduler...")

	// 取消 context，通知所有 goroutine 停止
	s.cancel()

	// 等待所有 goroutine 完成
	s.wg.Wait()

	logrus.Info("scheduler stopped")
	return nil
}

// runJob 运行单个任务
func (s *Scheduler) runJob(job Job) {
	defer s.wg.Done()

	log := logrus.WithField("job", job.Name())
	log.Infof("job started with delay=%v, period=%v", job.DelayTime(), job.PeriodTime())

	// 初始延迟
	timer := time.NewTimer(job.DelayTime())
	defer timer.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Info("job stopped due to context cancellation")
			return
		case <-timer.C:
			// 执行任务
			log.Debug("executing job")
			job.Run(s.ctx)

			// 重置定时器
			timer.Reset(job.PeriodTime())
		}
	}
}
