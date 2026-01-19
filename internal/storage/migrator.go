package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"

	"cdk-get/internal/errors"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrator 数据库迁移工具
type Migrator struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewMigrator 创建新的迁移工具实例
func NewMigrator(db *sql.DB, logger *logrus.Logger) *Migrator {
	if logger == nil {
		logger = logrus.New()
	}
	return &Migrator{
		db:     db,
		logger: logger,
	}
}

// Migration 表示一个迁移
type Migration struct {
	Version int
	Name    string
	UpSQL   string
	DownSQL string
}

// Migrate 执行所有待执行的迁移
func (m *Migrator) Migrate(ctx context.Context) error {
	// 创建迁移历史表
	if err := m.createMigrationTable(ctx); err != nil {
		return err
	}

	// 获取所有迁移
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	// 获取已执行的迁移版本
	appliedVersions, err := m.getAppliedVersions(ctx)
	if err != nil {
		return err
	}

	// 执行待执行的迁移
	for _, migration := range migrations {
		if _, applied := appliedVersions[migration.Version]; applied {
			m.logger.WithFields(logrus.Fields{
				"version": migration.Version,
				"name":    migration.Name,
			}).Debug("migration already applied, skipping")
			continue
		}

		m.logger.WithFields(logrus.Fields{
			"version": migration.Version,
			"name":    migration.Name,
		}).Info("applying migration")

		if err := m.applyMigration(ctx, migration); err != nil {
			return errors.NewDatabaseError(fmt.Sprintf("apply_migration_%d", migration.Version), err)
		}

		m.logger.WithFields(logrus.Fields{
			"version": migration.Version,
			"name":    migration.Name,
		}).Info("migration applied successfully")
	}

	m.logger.Info("all migrations applied successfully")
	return nil
}

// createMigrationTable 创建迁移历史表
func (m *Migrator) createMigrationTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return errors.NewDatabaseError("create_migration_table", err)
	}

	return nil
}

// loadMigrations 从嵌入的文件系统加载所有迁移
func (m *Migrator) loadMigrations() ([]Migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// 按文件名分组迁移
	migrationMap := make(map[int]*Migration)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasSuffix(filename, ".sql") {
			continue
		}

		// 解析文件名: 000001_initial_schema.up.sql
		var version int
		var name, direction string
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) != 2 {
			m.logger.WithField("filename", filename).Warn("skipping invalid migration filename")
			continue
		}

		if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
			m.logger.WithFields(logrus.Fields{
				"filename": filename,
				"error":    err,
			}).Warn("failed to parse migration version")
			continue
		}

		// 提取名称和方向
		nameAndDirection := strings.TrimSuffix(parts[1], ".sql")
		lastDot := strings.LastIndex(nameAndDirection, ".")
		if lastDot == -1 {
			m.logger.WithField("filename", filename).Warn("skipping invalid migration filename format")
			continue
		}

		name = nameAndDirection[:lastDot]
		direction = nameAndDirection[lastDot+1:]

		// 读取SQL内容
		content, err := migrationsFS.ReadFile("migrations/" + filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// 获取或创建迁移
		migration, exists := migrationMap[version]
		if !exists {
			migration = &Migration{
				Version: version,
				Name:    name,
			}
			migrationMap[version] = migration
		}

		// 设置SQL内容
		if direction == "up" {
			migration.UpSQL = string(content)
		} else if direction == "down" {
			migration.DownSQL = string(content)
		}
	}

	// 转换为切片并排序
	migrations := make([]Migration, 0, len(migrationMap))
	for _, migration := range migrationMap {
		if migration.UpSQL == "" {
			m.logger.WithFields(logrus.Fields{
				"version": migration.Version,
				"name":    migration.Name,
			}).Warn("migration missing up SQL, skipping")
			continue
		}
		migrations = append(migrations, *migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// getAppliedVersions 获取已执行的迁移版本
func (m *Migrator) getAppliedVersions(ctx context.Context) (map[int]bool, error) {
	query := `SELECT version FROM schema_migrations`
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.NewDatabaseError("get_applied_versions", err)
	}
	defer rows.Close()

	versions := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, errors.NewDatabaseError("scan_version", err)
		}
		versions[version] = true
	}

	if err := rows.Err(); err != nil {
		return nil, errors.NewDatabaseError("iterate_versions", err)
	}

	return versions, nil
}

// applyMigration 执行单个迁移
func (m *Migrator) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 执行迁移SQL
	if _, err := tx.ExecContext(ctx, migration.UpSQL); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// 记录迁移历史
	recordQuery := `INSERT INTO schema_migrations (version, name) VALUES (?, ?)`
	if _, err := tx.ExecContext(ctx, recordQuery, migration.Version, migration.Name); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Rollback 回滚指定数量的迁移
func (m *Migrator) Rollback(ctx context.Context, steps int) error {
	// 获取已执行的迁移
	appliedVersions, err := m.getAppliedVersions(ctx)
	if err != nil {
		return err
	}

	// 加载所有迁移
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	// 找到需要回滚的迁移（按版本倒序）
	var toRollback []Migration
	for i := len(migrations) - 1; i >= 0 && len(toRollback) < steps; i-- {
		if _, applied := appliedVersions[migrations[i].Version]; applied {
			toRollback = append(toRollback, migrations[i])
		}
	}

	// 执行回滚
	for _, migration := range toRollback {
		if migration.DownSQL == "" {
			m.logger.WithFields(logrus.Fields{
				"version": migration.Version,
				"name":    migration.Name,
			}).Warn("migration has no down SQL, skipping rollback")
			continue
		}

		m.logger.WithFields(logrus.Fields{
			"version": migration.Version,
			"name":    migration.Name,
		}).Info("rolling back migration")

		if err := m.rollbackMigration(ctx, migration); err != nil {
			return errors.NewDatabaseError(fmt.Sprintf("rollback_migration_%d", migration.Version), err)
		}

		m.logger.WithFields(logrus.Fields{
			"version": migration.Version,
			"name":    migration.Name,
		}).Info("migration rolled back successfully")
	}

	return nil
}

// rollbackMigration 回滚单个迁移
func (m *Migrator) rollbackMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 执行回滚SQL
	if _, err := tx.ExecContext(ctx, migration.DownSQL); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	// 删除迁移历史记录
	deleteQuery := `DELETE FROM schema_migrations WHERE version = ?`
	if _, err := tx.ExecContext(ctx, deleteQuery, migration.Version); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete migration record: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
