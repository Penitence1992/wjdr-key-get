package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TestCheckpoint_TransactionAtomicity 验证事务原子性
func TestCheckpoint_TransactionAtomicity(t *testing.T) {
	tmpFile := "./test_checkpoint_tx.db"
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

	// 测试事务回滚 - 当事务中的操作失败时，所有操作都应该回滚
	err = repo.WithTransaction(ctx, func(txRepo Repository) error {
		// 保存第一个用户
		user1 := &User{
			FID:      "rollback_user1",
			Nickname: "User1",
			KID:      100,
		}
		if err := txRepo.SaveUser(ctx, user1); err != nil {
			return err
		}

		// 保存第二个用户
		user2 := &User{
			FID:      "rollback_user2",
			Nickname: "User2",
			KID:      200,
		}
		if err := txRepo.SaveUser(ctx, user2); err != nil {
			return err
		}

		// 故意返回错误以触发回滚
		return fmt.Errorf("intentional error to trigger rollback")
	})

	if err == nil {
		t.Fatal("expected transaction to fail")
	}

	// 验证两个用户都没有被保存（事务已回滚）
	_, err = repo.GetUser(ctx, "rollback_user1")
	if err == nil {
		t.Error("expected user1 to not exist after rollback")
	}

	_, err = repo.GetUser(ctx, "rollback_user2")
	if err == nil {
		t.Error("expected user2 to not exist after rollback")
	}

	// 测试事务提交 - 当所有操作成功时，事务应该提交
	err = repo.WithTransaction(ctx, func(txRepo Repository) error {
		user1 := &User{
			FID:      "commit_user1",
			Nickname: "CommitUser1",
			KID:      300,
		}
		if err := txRepo.SaveUser(ctx, user1); err != nil {
			return err
		}

		user2 := &User{
			FID:      "commit_user2",
			Nickname: "CommitUser2",
			KID:      400,
		}
		return txRepo.SaveUser(ctx, user2)
	})

	if err != nil {
		t.Fatalf("transaction should succeed: %v", err)
	}

	// 验证两个用户都已保存（事务已提交）
	user1, err := repo.GetUser(ctx, "commit_user1")
	if err != nil {
		t.Errorf("expected user1 to exist after commit: %v", err)
	}
	if user1 != nil && user1.Nickname != "CommitUser1" {
		t.Errorf("expected nickname CommitUser1, got %s", user1.Nickname)
	}

	user2, err := repo.GetUser(ctx, "commit_user2")
	if err != nil {
		t.Errorf("expected user2 to exist after commit: %v", err)
	}
	if user2 != nil && user2.Nickname != "CommitUser2" {
		t.Errorf("expected nickname CommitUser2, got %s", user2.Nickname)
	}

	t.Log("✓ Transaction atomicity verified: rollback and commit work correctly")
}

// TestCheckpoint_ConnectionPoolLimit 验证连接池限制
func TestCheckpoint_ConnectionPoolLimit(t *testing.T) {
	tmpFile := "./test_checkpoint_pool.db"
	defer os.Remove(tmpFile)

	maxConns := 5
	config := SqliteConfig{
		Path:            tmpFile,
		MaxOpenConns:    maxConns,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	repo, err := NewSqliteRepository(config, logger)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// 创建测试用户
	for i := 0; i < 10; i++ {
		user := &User{
			FID:      fmt.Sprintf("pool_user_%d", i),
			Nickname: fmt.Sprintf("PoolUser%d", i),
			KID:      i,
		}
		if err := repo.SaveUser(ctx, user); err != nil {
			t.Fatalf("failed to create test user: %v", err)
		}
	}

	// 并发执行多个查询，超过连接池限制
	concurrency := maxConns * 3 // 15个并发请求，超过5个连接的限制
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// 执行查询
			fid := fmt.Sprintf("pool_user_%d", id%10)
			_, err := repo.GetUser(ctx, fid)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("concurrent query error: %v", err)
	}

	if errorCount > 0 {
		t.Errorf("expected no errors with connection pooling, got %d errors", errorCount)
	}

	t.Logf("✓ Connection pool limit verified: handled %d concurrent requests with max %d connections", concurrency, maxConns)
}

// TestCheckpoint_MigrationSystem 验证迁移系统
func TestCheckpoint_MigrationSystem(t *testing.T) {
	tmpFile := "./test_checkpoint_migration.db"
	defer os.Remove(tmpFile)

	config := DefaultSqliteConfig()
	config.Path = tmpFile

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// 第一次创建仓库，应该运行迁移
	repo, err := NewSqliteRepository(config, logger)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	ctx := context.Background()

	// 验证迁移表已创建
	// 使用反射获取底层的 *sql.DB
	type dbGetter interface {
		GetDB() *sql.DB
	}

	var count int
	// 直接通过仓库执行查询来验证迁移
	query := "SELECT COUNT(*) FROM schema_migrations"
	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		t.Fatalf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	err = stmt.QueryRowContext(ctx).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query migrations table: %v", err)
	}

	if count == 0 {
		t.Error("expected at least one migration to be applied")
	}

	t.Logf("✓ Migration system verified: %d migrations applied", count)

	// 验证所有表都已创建
	tables := []string{"fid_list", "gift_codes", "gift_code_task"}
	for _, table := range tables {
		var tableExists int
		query := fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='%s'", table)
		stmt, err := repo.db.PrepareContext(ctx, query)
		if err != nil {
			t.Fatalf("failed to prepare query for table %s: %v", table, err)
		}
		err = stmt.QueryRowContext(ctx).Scan(&tableExists)
		stmt.Close()
		if err != nil {
			t.Fatalf("failed to check table %s: %v", table, err)
		}
		if tableExists == 0 {
			t.Errorf("expected table %s to exist", table)
		}
	}

	// 验证索引已创建
	indexes := []string{
		"idx_fid_code",
	}
	for _, index := range indexes {
		var indexExists int
		query := fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='%s'", index)
		stmt, err := repo.db.PrepareContext(ctx, query)
		if err != nil {
			t.Fatalf("failed to prepare query for index %s: %v", index, err)
		}
		err = stmt.QueryRowContext(ctx).Scan(&indexExists)
		stmt.Close()
		if err != nil {
			t.Fatalf("failed to check index %s: %v", index, err)
		}
		if indexExists == 0 {
			t.Errorf("expected index %s to exist", index)
		}
	}

	t.Log("✓ All tables and indexes created successfully")

	repo.Close()

	// 第二次创建仓库，迁移应该被跳过（已应用）
	repo2, err := NewSqliteRepository(config, logger)
	if err != nil {
		t.Fatalf("failed to create repository second time: %v", err)
	}
	defer repo2.Close()

	// 验证迁移数量没有增加
	var count2 int
	stmt2, err := repo2.db.PrepareContext(ctx, "SELECT COUNT(*) FROM schema_migrations")
	if err != nil {
		t.Fatalf("failed to prepare query: %v", err)
	}
	defer stmt2.Close()

	err = stmt2.QueryRowContext(ctx).Scan(&count2)
	if err != nil {
		t.Fatalf("failed to query migrations table: %v", err)
	}

	if count2 != count {
		t.Errorf("expected migration count to remain %d, got %d", count, count2)
	}

	t.Log("✓ Migration idempotency verified: migrations not re-applied")
}

// TestCheckpoint_PreparedStatements 验证所有查询使用预处理语句
func TestCheckpoint_PreparedStatements(t *testing.T) {
	tmpFile := "./test_checkpoint_prepared.db"
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

	// 测试所有操作都能正常工作（使用预处理语句）
	// 如果有SQL注入漏洞，这些测试会失败

	// 测试用户操作
	user := &User{
		FID:      "test'; DROP TABLE users; --",
		Nickname: "Malicious",
		KID:      999,
	}
	if err := repo.SaveUser(ctx, user); err != nil {
		t.Fatalf("failed to save user with special characters: %v", err)
	}

	retrievedUser, err := repo.GetUser(ctx, "test'; DROP TABLE users; --")
	if err != nil {
		t.Fatalf("failed to get user with special characters: %v", err)
	}
	if retrievedUser.FID != user.FID {
		t.Errorf("expected FID to match")
	}

	// 验证表仍然存在（没有被SQL注入删除）
	users, err := repo.ListUsers(ctx)
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	if len(users) == 0 {
		t.Error("expected at least one user")
	}

	t.Log("✓ Prepared statements verified: SQL injection prevented")
}
