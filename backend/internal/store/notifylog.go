package store

import (
	"monitor/internal/model"
)

// CreateNotificationLog 写入通知发送日志。
func (s *Store) CreateNotificationLog(l *model.NotificationLog) error {
	return s.db.Create(l).Error
}

// ListNotificationLogs 分页查询通知日志。
func (s *Store) ListNotificationLogs(page, size int) ([]model.NotificationLog, int64, error) {
	var list []model.NotificationLog
	var total int64
	q := s.db.Model(&model.NotificationLog{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}
