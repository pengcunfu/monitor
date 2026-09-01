package store

import (
	"errors"

	"gorm.io/gorm"

	"monitor/internal/model"
)

// ListChannels 返回全部通知渠道。
func (s *Store) ListChannels() ([]model.NotificationChannel, error) {
	var list []model.NotificationChannel
	err := s.db.Order("id ASC").Find(&list).Error
	return list, err
}

// GetChannel 按 ID 查询渠道。
func (s *Store) GetChannel(id uint) (*model.NotificationChannel, error) {
	var c model.NotificationChannel
	err := s.db.First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateChannel 新建渠道。
func (s *Store) CreateChannel(c *model.NotificationChannel) error {
	return s.db.Create(c).Error
}

// UpdateChannel 更新渠道。
func (s *Store) UpdateChannel(c *model.NotificationChannel) error {
	return s.db.Save(c).Error
}

// DeleteChannel 删除渠道。
func (s *Store) DeleteChannel(id uint) error {
	return s.db.Delete(&model.NotificationChannel{}, id).Error
}

// FindChannelByNameType 按名称与类型查询渠道（用于系统设置页 upsert 内置 SMTP 渠道）。
func (s *Store) FindChannelByNameType(name, typ string) (*model.NotificationChannel, error) {
	var c model.NotificationChannel
	err := s.db.Where("name = ? AND type = ?", name, typ).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListEnabledChannels 返回启用中的渠道（通知管理器缓存用）。
func (s *Store) ListEnabledChannels() ([]model.NotificationChannel, error) {
	var list []model.NotificationChannel
	err := s.db.Where("enabled = ?", true).Find(&list).Error
	return list, err
}
