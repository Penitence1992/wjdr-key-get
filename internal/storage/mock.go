package storage

import (
	"context"
	"time"
)

// MockKeyStorage 用于测试的KeyStorage mock实现
type MockKeyStorage struct{}

func (m *MockKeyStorage) IsReceived(fid, code string) (bool, error) {
	return false, nil
}

func (m *MockKeyStorage) Save(fid, code string) error {
	return nil
}

func (m *MockKeyStorage) GetFids() ([]string, error) {
	return []string{}, nil
}

func (m *MockKeyStorage) SaveFidInfo(fid int, nickname string, kid int, avatarImage string) error {
	return nil
}

func (m *MockKeyStorage) AddTask(code string) error {
	return nil
}

func (m *MockKeyStorage) GetTask() ([]string, error) {
	return []string{}, nil
}

func (m *MockKeyStorage) DoneTask(code string) error {
	return nil
}

// MockRepository 用于测试的Repository mock实现
type MockRepository struct {
	DeleteTaskFunc func(ctx context.Context, code string) error
}

func (m *MockRepository) SaveGiftCode(ctx context.Context, fid, code string) error {
	return nil
}

func (m *MockRepository) IsGiftCodeReceived(ctx context.Context, fid, code string) (bool, error) {
	return false, nil
}

func (m *MockRepository) ListGiftCodesByFID(ctx context.Context, fid string) ([]*GiftCodeRecord, error) {
	return []*GiftCodeRecord{}, nil
}

func (m *MockRepository) SaveUser(ctx context.Context, user *User) error {
	return nil
}

func (m *MockRepository) GetUser(ctx context.Context, fid string) (*User, error) {
	return nil, nil
}

func (m *MockRepository) ListUsers(ctx context.Context) ([]*User, error) {
	return []*User{}, nil
}

func (m *MockRepository) CreateTask(ctx context.Context, code string) error {
	return nil
}

func (m *MockRepository) ListPendingTasks(ctx context.Context) ([]*Task, error) {
	return []*Task{}, nil
}

func (m *MockRepository) MarkTaskComplete(ctx context.Context, code string) error {
	return nil
}

func (m *MockRepository) GetTaskByCode(ctx context.Context, code string) (*Task, error) {
	return nil, nil
}

func (m *MockRepository) UpdateTaskRetry(ctx context.Context, code string, retryCount int, lastError string) error {
	return nil
}

func (m *MockRepository) UpdateTaskComplete(ctx context.Context, code string, completedAt time.Time) error {
	return nil
}

func (m *MockRepository) ListCompletedTasks(ctx context.Context, limit int) ([]*Task, error) {
	return []*Task{}, nil
}

func (m *MockRepository) DeleteTask(ctx context.Context, code string) error {
	if m.DeleteTaskFunc != nil {
		return m.DeleteTaskFunc(ctx, code)
	}
	return nil
}

func (m *MockRepository) SaveNotification(ctx context.Context, notification *Notification) error {
	return nil
}

func (m *MockRepository) ListNotifications(ctx context.Context, limit int) ([]*Notification, error) {
	return []*Notification{}, nil
}

func (m *MockRepository) WithTransaction(ctx context.Context, fn func(Repository) error) error {
	return fn(m)
}

func (m *MockRepository) Ping(ctx context.Context) error {
	return nil
}

func (m *MockRepository) Close() error {
	return nil
}
