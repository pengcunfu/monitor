package store

import (
	"monitor/internal/model"
)

// InsertServiceStates 批量写入服务状态快照。
func (s *Store) InsertServiceStates(states []model.ServiceState) error {
	if len(states) == 0 {
		return nil
	}
	return s.db.CreateInBatches(states, 200).Error
}

// LatestServiceStates 返回最新一轮服务状态。
func (s *Store) LatestServiceStates() ([]model.ServiceState, error) {
	var latestTs struct{ Ts int64 }
	err := s.db.Model(&model.ServiceState{}).Select("MAX(ts) AS ts").Scan(&latestTs).Error
	if err != nil {
		return nil, err
	}
	if latestTs.Ts == 0 {
		return []model.ServiceState{}, nil
	}
	var out []model.ServiceState
	err = s.db.Where("ts = ?", latestTs.Ts).Order("name ASC").Find(&out).Error
	return out, err
}

// ServiceHistory 指定服务的状态变化历史。
func (s *Store) ServiceHistory(name string, from, to int64) ([]model.ServiceState, error) {
	var out []model.ServiceState
	err := s.db.Where("name = ? AND ts BETWEEN ? AND ?", name, from, to).
		Order("ts ASC").Find(&out).Error
	return out, err
}

// ServiceNames 返回采样中出现过的服务名（去重）。
func (s *Store) ServiceNames() ([]string, error) {
	var names []string
	err := s.db.Model(&model.ServiceState{}).Distinct("name").Order("name ASC").Pluck("name", &names).Error
	return names, err
}
