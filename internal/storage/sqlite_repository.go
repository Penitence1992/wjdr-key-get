package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/sirupsen/logrus"

	"cdk-get/internal/errors"
)

// SqliteRepository SQLite数据库仓库实现
type SqliteRepository struct {
	db     dbInterface
	logger *logrus.Logger
}

// SqliteConfig SQLite配置
type SqliteConfig struct {
	Path            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
}

// DefaultSqliteConfig 返回默认SQLite配置
func DefaultSqliteConfig() SqliteConfig {
	return SqliteConfig{
		Path:            "./giftcode.db",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
	}
}

// NewSqliteRepository 创建新的SQLite仓库实例
func NewSqliteRepository(config SqliteConfig, logger *logrus.Logger) (*SqliteRepository, error) {
	if logger == nil {
		logger = logrus.New()
	}

	// 使用指数退避重试连接
	var db *sql.DB
	var err error

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		db, err = sql.Open("sqlite3", config.Path)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}

		if attempt < config.MaxRetries-1 {
			delay := config.RetryDelay * time.Duration(1<<uint(attempt)) // 指数退避
			logger.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"delay":   delay,
				"error":   err,
			}).Warn("failed to connect to database, retrying...")
			time.Sleep(delay)
		}
	}

	if err != nil {
		return nil, errors.NewDatabaseError("connect", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	repo := &SqliteRepository{
		db:     &sqlDB{db: db},
		logger: logger,
	}

	// 运行数据库迁移
	migrator := NewMigrator(db, logger)
	if err := migrator.Migrate(context.Background()); err != nil {
		db.Close()
		return nil, errors.NewDatabaseError("run_migrations", err)
	}

	logger.WithFields(logrus.Fields{
		"path":              config.Path,
		"max_open_conns":    config.MaxOpenConns,
		"max_idle_conns":    config.MaxIdleConns,
		"conn_max_lifetime": config.ConnMaxLifetime,
	}).Info("sqlite repository initialized successfully")

	return repo, nil
}

// SaveGiftCode 保存礼品码记录
func (r *SqliteRepository) SaveGiftCode(ctx context.Context, fid, code string) error {
	query := `INSERT INTO gift_codes (fid, code) VALUES (?, ?)`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return errors.NewDatabaseError("prepare_save_gift_code", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, fid, code)
	if err != nil {
		return errors.NewDatabaseError("save_gift_code", err)
	}

	r.logger.WithFields(logrus.Fields{
		"fid":  fid,
		"code": code,
	}).Debug("gift code saved successfully")

	return nil
}

// IsGiftCodeReceived 检查礼品码是否已被领取
func (r *SqliteRepository) IsGiftCodeReceived(ctx context.Context, fid, code string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM gift_codes WHERE fid = ? AND code = ?)`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return false, errors.NewDatabaseError("prepare_check_gift_code", err)
	}
	defer stmt.Close()

	var exists bool
	err = stmt.QueryRowContext(ctx, fid, code).Scan(&exists)
	if err != nil {
		return false, errors.NewDatabaseError("check_gift_code", err)
	}

	return exists, nil
}

// ListGiftCodesByFID 列出用户的所有礼品码记录
func (r *SqliteRepository) ListGiftCodesByFID(ctx context.Context, fid string) ([]*GiftCodeRecord, error) {
	query := `SELECT id, fid, code FROM gift_codes WHERE fid = ? ORDER BY id DESC`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, errors.NewDatabaseError("prepare_list_gift_codes", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, fid)
	if err != nil {
		return nil, errors.NewDatabaseError("list_gift_codes", err)
	}
	defer rows.Close()

	var records []*GiftCodeRecord
	for rows.Next() {
		var record GiftCodeRecord
		err := rows.Scan(&record.ID, &record.FID, &record.Code)
		if err != nil {
			return nil, errors.NewDatabaseError("scan_gift_code", err)
		}
		// Set default values for fields not in gift_codes table
		record.Status = GiftCodeStatusSuccess
		record.Message = ""
		record.CreatedAt = time.Now() // Note: gift_codes table doesn't have created_at
		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.NewDatabaseError("iterate_gift_codes", err)
	}

	return records, nil
}

// SaveUser 保存或更新用户信息
func (r *SqliteRepository) SaveUser(ctx context.Context, user *User) error {
	// 检查用户是否存在
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM fid_list WHERE fid = ?)`
	checkStmt, err := r.db.PrepareContext(ctx, checkQuery)
	if err != nil {
		return errors.NewDatabaseError("prepare_check_user", err)
	}
	defer checkStmt.Close()

	err = checkStmt.QueryRowContext(ctx, user.FID).Scan(&exists)
	if err != nil {
		return errors.NewDatabaseError("check_user", err)
	}

	var query string
	var stmt *sql.Stmt

	if exists {
		// 更新现有用户
		query = `UPDATE fid_list SET nickname = ?, kid = ?, avatar_image = ? WHERE fid = ?`
		stmt, err = r.db.PrepareContext(ctx, query)
		if err != nil {
			return errors.NewDatabaseError("prepare_update_user", err)
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx, user.Nickname, user.KID, user.AvatarImage, user.FID)
		if err != nil {
			return errors.NewDatabaseError("update_user", err)
		}

		r.logger.WithFields(logrus.Fields{
			"fid": user.FID,
		}).Debug("user updated successfully")
	} else {
		// 插入新用户
		query = `INSERT INTO fid_list (fid, nickname, kid, avatar_image) VALUES (?, ?, ?, ?)`
		stmt, err = r.db.PrepareContext(ctx, query)
		if err != nil {
			return errors.NewDatabaseError("prepare_insert_user", err)
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx, user.FID, user.Nickname, user.KID, user.AvatarImage)
		if err != nil {
			return errors.NewDatabaseError("insert_user", err)
		}

		r.logger.WithFields(logrus.Fields{
			"fid": user.FID,
		}).Debug("user created successfully")
	}

	return nil
}

// GetUser 获取用户信息
func (r *SqliteRepository) GetUser(ctx context.Context, fid string) (*User, error) {
	query := `SELECT fid, nickname, kid, avatar_image FROM fid_list WHERE fid = ?`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, errors.NewDatabaseError("prepare_get_user", err)
	}
	defer stmt.Close()

	var user User
	err = stmt.QueryRowContext(ctx, fid).Scan(
		&user.FID,
		&user.Nickname,
		&user.KID,
		&user.AvatarImage,
	)
	if err == sql.ErrNoRows {
		return nil, errors.NewNotFoundError("user", fid)
	}
	if err != nil {
		return nil, errors.NewDatabaseError("get_user", err)
	}

	// Set default timestamps since fid_list doesn't have these fields
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	return &user, nil
}

// ListUsers 列出所有用户
func (r *SqliteRepository) ListUsers(ctx context.Context) ([]*User, error) {
	query := `SELECT fid, nickname, kid, avatar_image FROM fid_list ORDER BY fid DESC`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, errors.NewDatabaseError("prepare_list_users", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, errors.NewDatabaseError("list_users", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.FID,
			&user.Nickname,
			&user.KID,
			&user.AvatarImage,
		)
		if err != nil {
			return nil, errors.NewDatabaseError("scan_user", err)
		}
		// Set default timestamps since fid_list doesn't have these fields
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.NewDatabaseError("iterate_users", err)
	}

	return users, nil
}

// CreateTask 创建新任务
func (r *SqliteRepository) CreateTask(ctx context.Context, code string) error {
	// Check if task already exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM gift_code_task WHERE code = ?)`
	checkStmt, err := r.db.PrepareContext(ctx, checkQuery)
	if err != nil {
		return errors.NewDatabaseError("prepare_check_task", err)
	}
	defer checkStmt.Close()

	err = checkStmt.QueryRowContext(ctx, code).Scan(&exists)
	if err != nil {
		return errors.NewDatabaseError("check_task", err)
	}

	if !exists {
		query := `INSERT INTO gift_code_task (code, all_done, created_at) VALUES (?, 0, CURRENT_TIMESTAMP)`
		stmt, err := r.db.PrepareContext(ctx, query)
		if err != nil {
			return errors.NewDatabaseError("prepare_create_task", err)
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx, code)
		if err != nil {
			return errors.NewDatabaseError("create_task", err)
		}

		r.logger.WithFields(logrus.Fields{
			"code": code,
		}).Debug("task created successfully")
	}

	return nil
}

// ListPendingTasks 列出所有待处理任务
func (r *SqliteRepository) ListPendingTasks(ctx context.Context) ([]*Task, error) {
	query := `SELECT code, all_done, retry_count, last_error, created_at, completed_at 
	          FROM gift_code_task 
	          WHERE all_done = 0 
	          ORDER BY code ASC`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, errors.NewDatabaseError("prepare_list_pending_tasks", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, errors.NewDatabaseError("list_pending_tasks", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var task Task
		var allDone int
		var createdAt sql.NullTime
		var completedAt sql.NullTime

		err := rows.Scan(
			&task.Code,
			&allDone,
			&task.RetryCount,
			&task.LastError,
			&createdAt,
			&completedAt,
		)
		if err != nil {
			return nil, errors.NewDatabaseError("scan_task", err)
		}

		task.AllDone = allDone != 0
		if createdAt.Valid {
			task.CreatedAt = createdAt.Time
			task.UpdatedAt = createdAt.Time
		} else {
			// Set default time if NULL
			task.CreatedAt = time.Now()
			task.UpdatedAt = time.Now()
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.NewDatabaseError("iterate_tasks", err)
	}

	return tasks, nil
}

// MarkTaskComplete 标记任务为完成
func (r *SqliteRepository) MarkTaskComplete(ctx context.Context, code string) error {
	query := `UPDATE gift_code_task SET all_done = 1 WHERE code = ?`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return errors.NewDatabaseError("prepare_mark_task_complete", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, code)
	if err != nil {
		return errors.NewDatabaseError("mark_task_complete", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewDatabaseError("get_rows_affected", err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("task", code)
	}

	r.logger.WithFields(logrus.Fields{
		"code":          code,
		"rows_affected": rowsAffected,
	}).Debug("task marked as complete")

	return nil
}

// UpdateTaskRetry 更新任务重试次数和错误信息
func (r *SqliteRepository) UpdateTaskRetry(ctx context.Context, code string, retryCount int, lastError string) error {
	query := `UPDATE gift_code_task SET retry_count = ?, last_error = ? WHERE code = ?`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"code":  code,
			"error": err,
		}).Error("failed to prepare update task retry statement")
		return errors.NewDatabaseError("prepare_update_task_retry", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, retryCount, lastError, code)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"code":        code,
			"retry_count": retryCount,
			"error":       err,
		}).Error("failed to update task retry")
		return errors.NewDatabaseError("update_task_retry", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewDatabaseError("get_rows_affected", err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("task", code)
	}

	r.logger.WithFields(logrus.Fields{
		"code":          code,
		"retry_count":   retryCount,
		"rows_affected": rowsAffected,
	}).Debug("task retry count updated successfully")

	return nil
}

// UpdateTaskComplete 更新任务完成状态和完成时间
func (r *SqliteRepository) UpdateTaskComplete(ctx context.Context, code string, completedAt time.Time) error {
	query := `UPDATE gift_code_task SET all_done = 1, completed_at = ? WHERE code = ?`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"code":  code,
			"error": err,
		}).Error("failed to prepare update task complete statement")
		return errors.NewDatabaseError("prepare_update_task_complete", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, completedAt, code)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"code":  code,
			"error": err,
		}).Error("failed to update task complete")
		return errors.NewDatabaseError("update_task_complete", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewDatabaseError("get_rows_affected", err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("task", code)
	}

	r.logger.WithFields(logrus.Fields{
		"code":          code,
		"completed_at":  completedAt,
		"rows_affected": rowsAffected,
	}).Debug("task marked as complete with timestamp")

	return nil
}

// ListCompletedTasks 列出已完成的任务
func (r *SqliteRepository) ListCompletedTasks(ctx context.Context, limit int) ([]*Task, error) {
	query := `SELECT code, all_done, retry_count, last_error, created_at, completed_at 
	          FROM gift_code_task 
	          WHERE all_done = 1 
	          ORDER BY completed_at DESC 
	          LIMIT ?`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("failed to prepare list completed tasks statement")
		return nil, errors.NewDatabaseError("prepare_list_completed_tasks", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, limit)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("failed to list completed tasks")
		return nil, errors.NewDatabaseError("list_completed_tasks", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var task Task
		var allDone int
		var createdAt sql.NullTime
		var completedAt sql.NullTime

		err := rows.Scan(
			&task.Code,
			&allDone,
			&task.RetryCount,
			&task.LastError,
			&createdAt,
			&completedAt,
		)
		if err != nil {
			r.logger.WithFields(logrus.Fields{
				"error": err,
			}).Error("failed to scan completed task")
			return nil, errors.NewDatabaseError("scan_completed_task", err)
		}

		task.AllDone = allDone != 0
		if createdAt.Valid {
			task.CreatedAt = createdAt.Time
			task.UpdatedAt = createdAt.Time
		} else {
			// Set default time if NULL
			task.CreatedAt = time.Now()
			task.UpdatedAt = time.Now()
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.NewDatabaseError("iterate_completed_tasks", err)
	}

	r.logger.WithFields(logrus.Fields{
		"count": len(tasks),
		"limit": limit,
	}).Debug("completed tasks listed successfully")

	return tasks, nil
}

// SaveNotification 保存通知记录
func (r *SqliteRepository) SaveNotification(ctx context.Context, notification *Notification) error {
	// 验证 status 字段值
	if notification.Status != NotificationStatusSuccess && notification.Status != NotificationStatusFailed {
		r.logger.WithFields(logrus.Fields{
			"status": notification.Status,
		}).Error("invalid notification status")
		return errors.NewValidationError("status", "must be 'success' or 'failed'")
	}

	query := `INSERT INTO notifications (channel, title, content, result, status, created_at) 
	          VALUES (?, ?, ?, ?, ?, ?)`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("failed to prepare save notification statement")
		return errors.NewDatabaseError("prepare_save_notification", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx,
		notification.Channel,
		notification.Title,
		notification.Content,
		notification.Result,
		notification.Status,
		notification.CreatedAt,
	)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"channel": notification.Channel,
			"title":   notification.Title,
			"error":   err,
		}).Error("failed to save notification")
		return errors.NewDatabaseError("save_notification", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.NewDatabaseError("get_notification_id", err)
	}

	notification.ID = id

	r.logger.WithFields(logrus.Fields{
		"id":      id,
		"channel": notification.Channel,
		"title":   notification.Title,
		"status":  notification.Status,
	}).Debug("notification saved successfully")

	return nil
}

// ListNotifications 列出通知记录
func (r *SqliteRepository) ListNotifications(ctx context.Context, limit int) ([]*Notification, error) {
	query := `SELECT id, channel, title, content, result, status, created_at 
	          FROM notifications 
	          ORDER BY created_at DESC 
	          LIMIT ?`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("failed to prepare list notifications statement")
		return nil, errors.NewDatabaseError("prepare_list_notifications", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, limit)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("failed to list notifications")
		return nil, errors.NewDatabaseError("list_notifications", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		var notification Notification

		err := rows.Scan(
			&notification.ID,
			&notification.Channel,
			&notification.Title,
			&notification.Content,
			&notification.Result,
			&notification.Status,
			&notification.CreatedAt,
		)
		if err != nil {
			r.logger.WithFields(logrus.Fields{
				"error": err,
			}).Error("failed to scan notification")
			return nil, errors.NewDatabaseError("scan_notification", err)
		}

		notifications = append(notifications, &notification)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.NewDatabaseError("iterate_notifications", err)
	}

	r.logger.WithFields(logrus.Fields{
		"count": len(notifications),
		"limit": limit,
	}).Debug("notifications listed successfully")

	return notifications, nil
}

// GetTaskByCode 获取任务信息
func (r *SqliteRepository) GetTaskByCode(ctx context.Context, code string) (*Task, error) {
	query := `SELECT code, all_done, retry_count, last_error, created_at, completed_at FROM gift_code_task WHERE code = ?`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, errors.NewDatabaseError("prepare_get_task", err)
	}
	defer stmt.Close()

	var task Task
	var allDone int
	var createdAt sql.NullTime
	var completedAt sql.NullTime

	err = stmt.QueryRowContext(ctx, code).Scan(
		&task.Code,
		&allDone,
		&task.RetryCount,
		&task.LastError,
		&createdAt,
		&completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.NewNotFoundError("task", code)
	}
	if err != nil {
		return nil, errors.NewDatabaseError("get_task", err)
	}

	task.AllDone = allDone != 0
	if createdAt.Valid {
		task.CreatedAt = createdAt.Time
		task.UpdatedAt = createdAt.Time
	} else {
		// Set default time if NULL
		task.CreatedAt = time.Now()
		task.UpdatedAt = time.Now()
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}

	return &task, nil
}

// WithTransaction 在事务中执行操作
func (r *SqliteRepository) WithTransaction(ctx context.Context, fn func(Repository) error) error {
	// 获取底层的 *sql.DB
	sqlDB, ok := r.db.(*sqlDB)
	if !ok {
		return errors.NewInternalError("cannot start transaction on non-db connection", nil)
	}

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return errors.NewDatabaseError("begin_transaction", err)
	}

	// 创建事务仓库
	txRepo := &SqliteRepository{
		db:     &txDB{tx: tx},
		logger: r.logger,
	}

	// 执行事务函数
	err = fn(txRepo)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			r.logger.WithFields(logrus.Fields{
				"error":          err,
				"rollback_error": rbErr,
			}).Error("failed to rollback transaction")
		}
		return err
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return errors.NewDatabaseError("commit_transaction", err)
	}

	return nil
}

// txDB 包装 sql.Tx 以实现 dbInterface
type txDB struct {
	tx *sql.Tx
}

func (t *txDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return t.tx.PrepareContext(ctx, query)
}

func (t *txDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *txDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

func (t *txDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

func (t *txDB) SetMaxOpenConns(n int)              {}
func (t *txDB) SetMaxIdleConns(n int)              {}
func (t *txDB) SetConnMaxLifetime(d time.Duration) {}

// Ping 检查数据库连接
func (r *SqliteRepository) Ping(ctx context.Context) error {
	if pinger, ok := r.db.(*sqlDB); ok {
		if err := pinger.PingContext(ctx); err != nil {
			return errors.NewDatabaseError("ping", err)
		}
	}
	return nil
}

// Close 关闭数据库连接
func (r *SqliteRepository) Close() error {
	if closer, ok := r.db.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			return errors.NewDatabaseError("close", err)
		}
	}
	r.logger.Info("sqlite repository closed successfully")
	return nil
}

// dbInterface 定义数据库操作接口，用于支持事务
type dbInterface interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// sqlDB 包装 *sql.DB
type sqlDB struct {
	db *sql.DB
}

func (s *sqlDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return s.db.PrepareContext(ctx, query)
}

func (s *sqlDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

func (s *sqlDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

func (s *sqlDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

func (s *sqlDB) Close() error {
	return s.db.Close()
}

func (s *sqlDB) PingContext(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *sqlDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, opts)
}

// KeyStorage interface implementation for backward compatibility
// These methods wrap the context-aware methods with a background context

// IsReceived 检查code是否已经获取过
func (r *SqliteRepository) IsReceived(fid, code string) (bool, error) {
	return r.IsGiftCodeReceived(context.Background(), fid, code)
}

// Save 保存获取记录
func (r *SqliteRepository) Save(fid, code string) error {
	return r.SaveGiftCode(context.Background(), fid, code)
}

// GetFids 获取用户id列表
func (r *SqliteRepository) GetFids() ([]string, error) {
	users, err := r.ListUsers(context.Background())
	if err != nil {
		return nil, err
	}

	fids := make([]string, 0, len(users))
	for _, user := range users {
		fids = append(fids, user.FID)
	}
	return fids, nil
}

// SaveFidInfo 保存用户信息
func (r *SqliteRepository) SaveFidInfo(fid int, nickname string, kid int, avatarImage string) error {
	user := &User{
		FID:         fmt.Sprintf("%d", fid),
		Nickname:    nickname,
		KID:         kid,
		AvatarImage: avatarImage,
	}
	return r.SaveUser(context.Background(), user)
}

// AddTask 新增任务
func (r *SqliteRepository) AddTask(code string) error {
	return r.CreateTask(context.Background(), code)
}

// GetTask 获取未完成的任务
func (r *SqliteRepository) GetTask() ([]string, error) {
	tasks, err := r.ListPendingTasks(context.Background())
	if err != nil {
		return nil, err
	}

	codes := make([]string, 0, len(tasks))
	for _, task := range tasks {
		codes = append(codes, task.Code)
	}
	return codes, nil
}

// DoneTask 完成任务
func (r *SqliteRepository) DoneTask(code string) error {
	return r.MarkTaskComplete(context.Background(), code)
}

// DeleteTask 删除任务及其关联的兑换码
// 在事务中执行，确保原子性
func (r *SqliteRepository) DeleteTask(ctx context.Context, code string) error {
	// 记录删除操作开始
	r.logger.WithFields(logrus.Fields{
		"code":      code,
		"operation": "DeleteTask",
		"stage":     "start",
	}).Info("starting delete task operation")

	// 获取底层的 *sql.DB 以开始事务
	sqlDB, ok := r.db.(*sqlDB)
	if !ok {
		r.logger.WithFields(logrus.Fields{
			"code":       code,
			"operation":  "DeleteTask",
			"error_type": "type_assertion_failed",
		}).Error("cannot start transaction on non-db connection")
		return fmt.Errorf("cannot start transaction on non-db connection")
	}

	// 开始事务
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"code":       code,
			"operation":  "DeleteTask",
			"stage":      "begin_transaction",
			"error_type": "transaction_begin_failed",
			"error":      err.Error(),
		}).Error("failed to begin transaction for delete task")
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // 如果没有提交，自动回滚

	// 注意：当前 gift_codes 表没有 task_id 字段，无法建立关联
	// 这里只删除 gift_codes 表中 code 字段匹配的记录
	// 如果需要真正的级联删除，需要先添加 task_id 字段到 gift_codes 表
	result, err := tx.ExecContext(ctx,
		"DELETE FROM gift_codes WHERE code = ?",
		code)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"code":       code,
			"operation":  "DeleteTask",
			"stage":      "delete_gift_codes",
			"error_type": "delete_failed",
			"error":      err.Error(),
			"table":      "gift_codes",
		}).Error("failed to delete gift codes")
		return fmt.Errorf("failed to delete gift codes: %w", err)
	}

	// 记录删除的兑换码数量
	giftCodesDeleted, _ := result.RowsAffected()
	r.logger.WithFields(logrus.Fields{
		"code":               code,
		"operation":          "DeleteTask",
		"stage":              "delete_gift_codes",
		"gift_codes_deleted": giftCodesDeleted,
	}).Debug("gift codes deleted")

	// 删除任务记录
	result, err = tx.ExecContext(ctx,
		"DELETE FROM gift_code_task WHERE code = ?",
		code)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"code":       code,
			"operation":  "DeleteTask",
			"stage":      "delete_task",
			"error_type": "delete_failed",
			"error":      err.Error(),
			"table":      "gift_code_task",
		}).Error("failed to delete task")
		return fmt.Errorf("failed to delete task: %w", err)
	}

	// 检查任务是否存在
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"code":       code,
			"operation":  "DeleteTask",
			"stage":      "check_rows_affected",
			"error_type": "rows_affected_failed",
			"error":      err.Error(),
		}).Error("failed to get rows affected")
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		r.logger.WithFields(logrus.Fields{
			"code":       code,
			"operation":  "DeleteTask",
			"stage":      "check_existence",
			"error_type": "not_found",
		}).Warn("task not found for deletion")
		return ErrTaskNotFound
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		r.logger.WithFields(logrus.Fields{
			"code":       code,
			"operation":  "DeleteTask",
			"stage":      "commit_transaction",
			"error_type": "commit_failed",
			"error":      err.Error(),
		}).Error("failed to commit transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 记录成功删除
	r.logger.WithFields(logrus.Fields{
		"code":               code,
		"operation":          "DeleteTask",
		"stage":              "complete",
		"result":             "success",
		"tasks_deleted":      rowsAffected,
		"gift_codes_deleted": giftCodesDeleted,
	}).Info("task deleted successfully")

	return nil
}
