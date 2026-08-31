package store

import (
	"errors"

	"gorm.io/gorm"

	"monitor/internal/model"
)

// InsertSnapshot 写入一条指标快照。
func (s *Store) InsertSnapshot(snap *model.MetricSnapshot) error {
	return s.db.Create(snap).Error
}

// LatestSnapshot 返回最新一条快照；无数据时返回 nil。
func (s *Store) LatestSnapshot() (*model.MetricSnapshot, error) {
	var snap model.MetricSnapshot
	err := s.db.Order("ts DESC").First(&snap).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

// SnapshotCount 快照总数（保留清理辅助）。
func (s *Store) SnapshotCount() (int64, error) {
	var n int64
	err := s.db.Model(&model.MetricSnapshot{}).Count(&n).Error
	return n, err
}

// SnapshotHistory 返回 [from, to] 时间范围内的快照（按 ts 升序）。
func (s *Store) SnapshotHistory(from, to int64) ([]model.MetricSnapshot, error) {
	var out []model.MetricSnapshot
	err := s.db.Where("ts BETWEEN ? AND ?", from, to).Order("ts ASC").Find(&out).Error
	return out, err
}
