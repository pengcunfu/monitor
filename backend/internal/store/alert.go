package store

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"monitor/internal/model"
)

// nowMs 当前 UTC 毫秒时间戳。
func nowMs() int64 { return time.Now().UnixMilli() }

// CreateAlertEvent 写入告警事件。
func (s *Store) CreateAlertEvent(ev *model.AlertEvent) error {
	return s.db.Create(ev).Error
}

// GetAlertEvent 按 ID 查询事件。
func (s *Store) GetAlertEvent(id uint) (*model.AlertEvent, error) {
	var ev model.AlertEvent
	err := s.db.First(&ev, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

// UpdateAlertEvent 更新事件。
func (s *Store) UpdateAlertEvent(ev *model.AlertEvent) error {
	return s.db.Save(ev).Error
}

// ListAlertEvents 分页查询事件，支持状态过滤与时间范围。
func (s *Store) ListAlertEvents(status string, from, to int64, page, size int) ([]model.AlertEvent, int64, error) {
	var list []model.AlertEvent
	var total int64
	q := s.db.Model(&model.AlertEvent{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if from > 0 {
		q = q.Where("fired_at >= ?", from)
	}
	if to > 0 {
		q = q.Where("fired_at <= ?", to)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("fired_at DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// FiringCount 当前处于 firing 状态的事件数。
func (s *Store) FiringCount() (int64, error) {
	var n int64
	err := s.db.Model(&model.AlertEvent{}).
		Where("status = ?", model.EventFiring).Count(&n).Error
	return n, err
}

// FiringEvents 当前 firing 事件（最新 N 条）。
func (s *Store) FiringEvents(limit int) ([]model.AlertEvent, error) {
	var list []model.AlertEvent
	err := s.db.Where("status = ?", model.EventFiring).
		Order("fired_at DESC").Limit(limit).Find(&list).Error
	return list, err
}

// AckAlert 确认告警事件。
func (s *Store) AckAlert(id uint, ackBy string) error {
	return s.db.Model(&model.AlertEvent{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"acked":  true,
			"ack_by": ackBy,
			"ack_at": nowMs(),
		}).Error
}

// AlertStats 返回指定时间段内的触发/恢复数量。
func (s *Store) AlertStats(from, to int64) (fired, resolved int64, err error) {
	if err = s.db.Model(&model.AlertEvent{}).
		Where("fired_at BETWEEN ? AND ?", from, to).Count(&fired).Error; err != nil {
		return
	}
	if err = s.db.Model(&model.AlertEvent{}).
		Where("resolved_at BETWEEN ? AND ? AND resolved_at > 0", from, to).Count(&resolved).Error; err != nil {
		return
	}
	return
}
