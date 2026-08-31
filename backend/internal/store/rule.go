package store

import (
	"errors"

	"gorm.io/gorm"

	"monitor/internal/model"
)

// ListEnabledRules 返回全部启用中的规则（引擎缓存用）。
func (s *Store) ListEnabledRules() ([]model.AlertRule, error) {
	var rules []model.AlertRule
	err := s.db.Where("enabled = ?", true).Find(&rules).Error
	return rules, err
}

// ListRules 分页查询规则。
func (s *Store) ListRules(page, size int) ([]model.AlertRule, int64, error) {
	var rules []model.AlertRule
	var total int64
	q := s.db.Model(&model.AlertRule{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := s.db.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rules).Error
	return rules, total, err
}

// GetRule 按 ID 查询规则。
func (s *Store) GetRule(id uint) (*model.AlertRule, error) {
	var r model.AlertRule
	err := s.db.First(&r, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateRule 新建规则。
func (s *Store) CreateRule(r *model.AlertRule) error {
	return s.db.Create(r).Error
}

// UpdateRule 更新规则。
func (s *Store) UpdateRule(r *model.AlertRule) error {
	return s.db.Save(r).Error
}

// DeleteRule 删除规则。
func (s *Store) DeleteRule(id uint) error {
	return s.db.Delete(&model.AlertRule{}, id).Error
}

// ToggleRule 启用/停用规则。
func (s *Store) ToggleRule(id uint, enabled bool) error {
	return s.db.Model(&model.AlertRule{}).Where("id = ?", id).
		Update("enabled", enabled).Error
}
