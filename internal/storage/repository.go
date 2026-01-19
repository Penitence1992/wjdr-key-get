package storage

import (
	"context"
	"errors"
	"time"
)

// Repository 数据仓库接口
type Repository interface {
	// Gift Code operations
	SaveGiftCode(ctx context.Context, fid, code string) error
	IsGiftCodeReceived(ctx context.Context, fid, code string) (bool, error)
	ListGiftCodesByFID(ctx context.Context, fid string) ([]*GiftCodeRecord, error)

	// User operations
	SaveUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, fid string) (*User, error)
	ListUsers(ctx context.Context) ([]*User, error)

	// Task operations
	CreateTask(ctx context.Context, code string) error
	ListPendingTasks(ctx context.Context) ([]*Task, error)
	MarkTaskComplete(ctx context.Context, code string) error
	GetTaskByCode(ctx context.Context, code string) (*Task, error)
	UpdateTaskRetry(ctx context.Context, code string, retryCount int, lastError string) error
	UpdateTaskComplete(ctx context.Context, code string, completedAt time.Time) error
	ListCompletedTasks(ctx context.Context, limit int) ([]*Task, error)
	// DeleteTask 删除任务及其关联的兑换码
	// 在事务中执行，确保原子性
	// 如果任务不存在，返回 ErrTaskNotFound
	DeleteTask(ctx context.Context, code string) error

	// Notification operations
	SaveNotification(ctx context.Context, notification *Notification) error
	ListNotifications(ctx context.Context, limit int) ([]*Notification, error)

	// Transaction support
	WithTransaction(ctx context.Context, fn func(Repository) error) error

	// Health check
	Ping(ctx context.Context) error
	Close() error
}

// User 用户模型
type User struct {
	FID         string    `json:"fid"`
	Nickname    string    `json:"nickname"`
	KID         int       `json:"kid"`
	AvatarImage string    `json:"avatar_image"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Task 任务模型
type Task struct {
	Code        string     `json:"code"`
	AllDone     bool       `json:"all_done"`
	RetryCount  int        `json:"retry_count"`
	LastError   string     `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// GiftCodeRecord 礼品码记录模型
type GiftCodeRecord struct {
	ID        int64     `json:"id"`
	FID       string    `json:"fid"`
	Code      string    `json:"code"`
	Status    string    `json:"status"` // success, failed, duplicate
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// GiftCodeStatus 礼品码状态常量
const (
	GiftCodeStatusSuccess   = "success"
	GiftCodeStatusFailed    = "failed"
	GiftCodeStatusDuplicate = "duplicate"
)

// Notification 通知记录模型
type Notification struct {
	ID        int64     `json:"id"`
	Channel   string    `json:"channel"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Result    string    `json:"result"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// NotificationStatus 通知状态常量
const (
	NotificationStatusSuccess = "success"
	NotificationStatusFailed  = "failed"
)

// ErrTaskNotFound 任务不存在错误
var ErrTaskNotFound = errors.New("task not found")
