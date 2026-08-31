package store

import (
	"monitor/internal/model"
)

// 全局设置 key。
const (
	SettingCollectIntervalSec     = "collect_interval_sec"      // 主指标采集周期（秒）
	SettingProcessIntervalSec     = "process_interval_sec"      // 进程采样周期（秒）
	SettingServiceIntervalSec     = "service_interval_sec"      // 服务状态采集周期（秒）
	SettingProcessTopN            = "process_top_n"             // 保留进程 top N
	SettingSnapshotRetainDays     = "snapshot_retain_days"      // 指标快照保留天数
	SettingProcessRetainDays      = "process_retain_days"       // 进程采样保留天数
	SettingServiceRetainDays      = "service_retain_days"       // 服务状态保留天数
	SettingAlertRetainDays        = "alert_retain_days"         // 告警事件保留天数
	SettingNotifyLogRetainDays    = "notify_log_retain_days"    // 通知日志保留天数
)

// DefaultSettings 各设置项的默认值（会被 config 覆盖后写入）。
func DefaultSettings() map[string]interface{} {
	return map[string]interface{}{
		SettingCollectIntervalSec:  10,
		SettingProcessIntervalSec:  30,
		SettingServiceIntervalSec:  30,
		SettingProcessTopN:         20,
		SettingSnapshotRetainDays:  7,
		SettingProcessRetainDays:   2,
		SettingServiceRetainDays:   3,
		SettingAlertRetainDays:     90,
		SettingNotifyLogRetainDays: 30,
	}
}

// EnsureDefaultSettings 为缺失的 key 写入默认值（不覆盖已有值）。
func (s *Store) EnsureDefaultSettings(defs map[string]interface{}) error {
	for k, v := range defs {
		var count int64
		if err := s.db.Model(&model.Setting{}).Where("key = ?", k).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := s.setSetting(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) setSetting(key string, value interface{}) error {
	var j model.JSON
	if err := j.Set(value); err != nil {
		return err
	}
	return s.db.Create(&model.Setting{Key: key, ValueJSON: j}).Error
}

// SetSetting 写入（或更新）单个设置。
func (s *Store) SetSetting(key string, value interface{}) error {
	var j model.JSON
	if err := j.Set(value); err != nil {
		return err
	}
	var st model.Setting
	err := s.db.Where("key = ?", key).First(&st).Error
	if err == nil {
		st.ValueJSON = j
		return s.db.Save(&st).Error
	}
	return s.db.Create(&model.Setting{Key: key, ValueJSON: j}).Error
}

// GetSettings 返回全部设置。
func (s *Store) GetSettings() (map[string]interface{}, error) {
	var rows []model.Setting
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]interface{}, len(rows))
	for _, r := range rows {
		var v interface{}
		if err := r.ValueJSON.Unmarshal(&v); err == nil {
			out[r.Key] = v
		}
	}
	return out, nil
}

// GetSetting 读取单个设置，不存在时返回默认值。
func (s *Store) GetSetting(key string, def interface{}) interface{} {
	var st model.Setting
	if err := s.db.Where("key = ?", key).First(&st).Error; err != nil {
		return def
	}
	var v interface{}
	if err := st.ValueJSON.Unmarshal(&v); err != nil {
		return def
	}
	return v
}

// GetSettingInt 读取整数设置。
func (s *Store) GetSettingInt(key string, def int) int {
	switch v := s.GetSetting(key, nil).(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return def
	}
}
