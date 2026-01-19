package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewSqliteRepository(t *testing.T) {
	// 创建临时数据库文件
	tmpFile := "./test_giftcode.db"
	defer os.Remove(tmpFile)

	config := SqliteConfig{
		Path:            tmpFile,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // 减少测试输出

	repo, err := NewSqliteRepository(config, logger)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	// 测试 Ping
	ctx := context.Background()
	if err := repo.Ping(ctx); err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

func TestSqliteRepository_SaveAndGetUser(t *testing.T) {
	tmpFile := "./test_user.db"
	defer os.Remove(tmpFile)

	config := DefaultSqliteConfig()
	config.Path = tmpFile

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	repo, err := NewSqliteRepository(config, logger)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// 测试保存用户
	user := &User{
		FID:         "test123",
		Nickname:    "TestUser",
		KID:         456,
		AvatarImage: "avatar.jpg",
	}

	if err := repo.SaveUser(ctx, user); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	// 测试获取用户
	retrievedUser, err := repo.GetUser(ctx, "test123")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}

	if retrievedUser.FID != user.FID {
		t.Errorf("expected FID %s, got %s", user.FID, retrievedUser.FID)
	}
	if retrievedUser.Nickname != user.Nickname {
		t.Errorf("expected Nickname %s, got %s", user.Nickname, retrievedUser.Nickname)
	}
}

func TestSqliteRepository_GiftCode(t *testing.T) {
	tmpFile := "./test_giftcode_ops.db"
	defer os.Remove(tmpFile)

	config := DefaultSqliteConfig()
	config.Path = tmpFile

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	repo, err := NewSqliteRepository(config, logger)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// 测试保存礼品码
	fid := "user123"
	code := "GIFT2024"

	if err := repo.SaveGiftCode(ctx, fid, code); err != nil {
		t.Fatalf("failed to save gift code: %v", err)
	}

	// 测试检查礼品码是否已领取
	received, err := repo.IsGiftCodeReceived(ctx, fid, code)
	if err != nil {
		t.Fatalf("failed to check gift code: %v", err)
	}
	if !received {
		t.Error("expected gift code to be received")
	}

	// 测试列出礼品码
	records, err := repo.ListGiftCodesByFID(ctx, fid)
	if err != nil {
		t.Fatalf("failed to list gift codes: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

func TestSqliteRepository_Task(t *testing.T) {
	tmpFile := "./test_task.db"
	defer os.Remove(tmpFile)

	config := DefaultSqliteConfig()
	config.Path = tmpFile

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	repo, err := NewSqliteRepository(config, logger)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// 测试创建任务
	code := "TASK2024"
	if err := repo.CreateTask(ctx, code); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 测试列出待处理任务
	tasks, err := repo.ListPendingTasks(ctx)
	if err != nil {
		t.Fatalf("failed to list pending tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// 测试标记任务完成
	if err := repo.MarkTaskComplete(ctx, code); err != nil {
		t.Fatalf("failed to mark task complete: %v", err)
	}

	// 验证任务已完成
	tasks, err = repo.ListPendingTasks(ctx)
	if err != nil {
		t.Fatalf("failed to list pending tasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 pending tasks, got %d", len(tasks))
	}
}

func TestSqliteRepository_Transaction(t *testing.T) {
	tmpFile := "./test_transaction.db"
	defer os.Remove(tmpFile)

	config := DefaultSqliteConfig()
	config.Path = tmpFile

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	repo, err := NewSqliteRepository(config, logger)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// 测试成功的事务
	err = repo.WithTransaction(ctx, func(txRepo Repository) error {
		user := &User{
			FID:      "tx_user",
			Nickname: "TxUser",
			KID:      789,
		}
		return txRepo.SaveUser(ctx, user)
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	// 验证用户已保存
	user, err := repo.GetUser(ctx, "tx_user")
	if err != nil {
		t.Fatalf("failed to get user after transaction: %v", err)
	}
	if user.Nickname != "TxUser" {
		t.Errorf("expected nickname TxUser, got %s", user.Nickname)
	}
}

func TestSqliteRepository_ConnectionRetry(t *testing.T) {
	// 测试连接重试逻辑
	config := SqliteConfig{
		Path:            "/invalid/path/test.db",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		MaxRetries:      2,
		RetryDelay:      10 * time.Millisecond,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	_, err := NewSqliteRepository(config, logger)
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestSqliteRepository_EnhancedTaskMethods(t *testing.T) {
	tmpFile := "./test_enhanced_task.db"
	defer os.Remove(tmpFile)

	config := DefaultSqliteConfig()
	config.Path = tmpFile

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	repo, err := NewSqliteRepository(config, logger)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// 创建测试任务
	code := "ENHANCED_TASK_2024"
	if err := repo.CreateTask(ctx, code); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 测试 UpdateTaskRetry
	if err := repo.UpdateTaskRetry(ctx, code, 1, "test error"); err != nil {
		t.Fatalf("failed to update task retry: %v", err)
	}

	// 验证重试次数已更新
	task, err := repo.GetTaskByCode(ctx, code)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if task.RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", task.RetryCount)
	}
	if task.LastError != "test error" {
		t.Errorf("expected last error 'test error', got '%s'", task.LastError)
	}

	// 测试 UpdateTaskComplete
	completedAt := time.Now()
	if err := repo.UpdateTaskComplete(ctx, code, completedAt); err != nil {
		t.Fatalf("failed to update task complete: %v", err)
	}

	// 验证任务已完成
	task, err = repo.GetTaskByCode(ctx, code)
	if err != nil {
		t.Fatalf("failed to get task after completion: %v", err)
	}
	if !task.AllDone {
		t.Error("expected task to be marked as done")
	}
	if task.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}

	// 测试 ListCompletedTasks
	completedTasks, err := repo.ListCompletedTasks(ctx, 10)
	if err != nil {
		t.Fatalf("failed to list completed tasks: %v", err)
	}
	if len(completedTasks) != 1 {
		t.Errorf("expected 1 completed task, got %d", len(completedTasks))
	}
	if completedTasks[0].Code != code {
		t.Errorf("expected code %s, got %s", code, completedTasks[0].Code)
	}
}

func TestSqliteRepository_NotificationMethods(t *testing.T) {
	tmpFile := "./test_notification.db"
	defer os.Remove(tmpFile)

	config := DefaultSqliteConfig()
	config.Path = tmpFile

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	repo, err := NewSqliteRepository(config, logger)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// 测试 SaveNotification - 成功通知
	notification := &Notification{
		Channel:   "wxpusher",
		Title:     "Test Notification",
		Content:   "This is a test notification",
		Result:    "Sent successfully",
		Status:    NotificationStatusSuccess,
		CreatedAt: time.Now(),
	}

	if err := repo.SaveNotification(ctx, notification); err != nil {
		t.Fatalf("failed to save notification: %v", err)
	}

	if notification.ID == 0 {
		t.Error("expected notification ID to be set")
	}

	// 测试 SaveNotification - 失败通知
	failedNotification := &Notification{
		Channel:   "wxpusher",
		Title:     "Failed Notification",
		Content:   "This notification failed",
		Result:    "Network error",
		Status:    NotificationStatusFailed,
		CreatedAt: time.Now(),
	}

	if err := repo.SaveNotification(ctx, failedNotification); err != nil {
		t.Fatalf("failed to save failed notification: %v", err)
	}

	// 测试无效状态
	invalidNotification := &Notification{
		Channel:   "wxpusher",
		Title:     "Invalid",
		Content:   "Invalid status",
		Result:    "N/A",
		Status:    "invalid_status",
		CreatedAt: time.Now(),
	}

	if err := repo.SaveNotification(ctx, invalidNotification); err == nil {
		t.Error("expected error for invalid status, got nil")
	}

	// 测试 ListNotifications
	notifications, err := repo.ListNotifications(ctx, 10)
	if err != nil {
		t.Fatalf("failed to list notifications: %v", err)
	}
	if len(notifications) != 2 {
		t.Errorf("expected 2 notifications, got %d", len(notifications))
	}

	// 验证排序（最新的在前）
	if notifications[0].Title != "Failed Notification" {
		t.Errorf("expected first notification to be 'Failed Notification', got '%s'", notifications[0].Title)
	}

	// 验证字段完整性
	for _, n := range notifications {
		if n.Channel == "" {
			t.Error("expected channel to be set")
		}
		if n.Title == "" {
			t.Error("expected title to be set")
		}
		if n.Status != NotificationStatusSuccess && n.Status != NotificationStatusFailed {
			t.Errorf("unexpected status: %s", n.Status)
		}
	}
}

func TestSqliteRepository_DeleteTask(t *testing.T) {
	tmpFile := "./test_delete_task.db"
	defer os.Remove(tmpFile)

	config := DefaultSqliteConfig()
	config.Path = tmpFile

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	repo, err := NewSqliteRepository(config, logger)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// 测试删除存在的任务（成功场景）
	t.Run("delete existing task", func(t *testing.T) {
		// 创建测试任务
		code := "DELETE_TEST_2024"
		if err := repo.CreateTask(ctx, code); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		// 添加一些关联的兑换码
		if err := repo.SaveGiftCode(ctx, "user1", code); err != nil {
			t.Fatalf("failed to save gift code: %v", err)
		}

		// 删除任务
		if err := repo.DeleteTask(ctx, code); err != nil {
			t.Fatalf("failed to delete task: %v", err)
		}

		// 验证任务已被删除
		_, err := repo.GetTaskByCode(ctx, code)
		if err == nil {
			t.Error("expected error when getting deleted task, got nil")
		}

		// 验证关联的兑换码也被删除
		received, err := repo.IsGiftCodeReceived(ctx, "user1", code)
		if err != nil {
			t.Fatalf("failed to check gift code: %v", err)
		}
		if received {
			t.Error("expected gift code to be deleted, but it still exists")
		}
	})

	// 测试删除不存在的任务（应返回 ErrTaskNotFound）
	t.Run("delete non-existent task", func(t *testing.T) {
		err := repo.DeleteTask(ctx, "NON_EXISTENT_TASK")
		if err != ErrTaskNotFound {
			t.Errorf("expected ErrTaskNotFound, got %v", err)
		}
	})

	// 测试删除没有关联兑换码的任务（边界情况）
	t.Run("delete task without gift codes", func(t *testing.T) {
		// 创建任务但不添加兑换码
		code := "TASK_NO_CODES"
		if err := repo.CreateTask(ctx, code); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		// 删除任务应该成功
		if err := repo.DeleteTask(ctx, code); err != nil {
			t.Fatalf("failed to delete task without gift codes: %v", err)
		}

		// 验证任务已被删除
		_, err := repo.GetTaskByCode(ctx, code)
		if err == nil {
			t.Error("expected error when getting deleted task, got nil")
		}
	})
}
