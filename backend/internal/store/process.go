package store

import (
	"time"

	"monitor/internal/model"
)

// InsertProcessSamples 批量写入进程采样（单事务）。
func (s *Store) InsertProcessSamples(samples []model.ProcessSample) error {
	if len(samples) == 0 {
		return nil
	}
	return s.db.CreateInBatches(samples, 200).Error
}

// LatestProcessSamples 返回最新一轮采样，按 cpu 或 mem 排序截断 topN。
func (s *Store) LatestProcessSamples(topN int, sortBy string) ([]model.ProcessSample, error) {
	var latestTs struct{ Ts int64 }
	err := s.db.Model(&model.ProcessSample{}).Select("MAX(ts) AS ts").Scan(&latestTs).Error
	if err != nil {
		return nil, err
	}
	if latestTs.Ts == 0 {
		return []model.ProcessSample{}, nil
	}
	order := "cpu_percent DESC"
	if sortBy == "mem" {
		order = "mem_percent DESC"
	}
	var out []model.ProcessSample
	err = s.db.Where("ts = ?", latestTs.Ts).Order(order).Limit(topN).Find(&out).Error
	return out, err
}

// ProcessHistory 指定进程名的历史曲线（最近 from 到 to，按 ts 升序）。
func (s *Store) ProcessHistory(name string, from, to int64) ([]model.ProcessSample, error) {
	var out []model.ProcessSample
	err := s.db.Where("name = ? AND ts BETWEEN ? AND ?", name, from, to).
		Order("ts ASC").Find(&out).Error
	return out, err
}

// ProcessNames 返回采样中出现过的进程名（去重，用于下拉选择）。
func (s *Store) ProcessNames() ([]string, error) {
	var names []string
	err := s.db.Model(&model.ProcessSample{}).
		Where("ts >= ?", time.Now().Add(-24*time.Hour).UnixMilli()).
		Distinct("name").Order("name ASC").Pluck("name", &names).Error
	return names, err
}
