package store

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"monitor/internal/model"
)

// Store 数据访问层：封装 GORM 与 SQLite。
type Store struct {
	db *gorm.DB
}

// Open 打开（或创建）SQLite 数据库并完成迁移与种子数据。
// 使用 WAL 模式与 busy_timeout 避免采集写入与 API 查询并发时互锁。
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据库目录失败: %w", err)
		}
	}
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("打开数据库 %s 失败: %w", path, err)
	}
	if err := db.AutoMigrate(model.AllModels()...); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	s := &Store{db: db}
	if err := s.seed(); err != nil {
		return nil, err
	}
	return s, nil
}

// DB 暴露底层 GORM 实例（采集器/告警引擎等需要直接操作）。
func (s *Store) DB() *gorm.DB { return s.db }

// Close 关闭数据库连接。
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// seed 初始化默认数据：管理员账号。
func (s *Store) seed() error {
	var count int64
	if err := s.db.Model(&model.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("生成初始密码哈希失败: %w", err)
		}
		admin := &model.User{Username: "admin", PasswordHash: string(hash), Role: "admin"}
		if err := s.db.Create(admin).Error; err != nil {
			return fmt.Errorf("创建默认管理员失败: %w", err)
		}
		log.Println("[store] 已创建默认管理员账号 admin / admin123，请尽快修改密码")
	}
	return nil
}
