package store

import (
	"log"
	"time"
)

// cleanupRule 表与对应时间列及保留天数。
type cleanupRule struct {
	table      string
	timeColumn string
	settingKey string
}

var cleanupRules = []cleanupRule{
	{"metric_snapshots", "ts", SettingSnapshotRetainDays},
	{"process_samples", "ts", SettingProcessRetainDays},
	{"service_states", "ts", SettingServiceRetainDays},
	{"alert_events", "fired_at", SettingAlertRetainDays},
	{"notification_logs", "created_at", SettingNotifyLogRetainDays},
}

// Cleanup 按保留策略分批删除过期历史数据。
func (s *Store) Cleanup() {
	for _, rule := range cleanupRules {
		retainDays := s.GetSettingInt(rule.settingKey, 7)
		if retainDays <= 0 {
			retainDays = 7
		}
		cutoff := time.Now().AddDate(0, 0, -retainDays).UnixMilli()
		// 分批删除（SQLite DELETE 不支持 LIMIT，用主键子查询限定批大小），避免大事务锁库
		for {
			res := s.db.Exec(
				"DELETE FROM "+rule.table+" WHERE id IN "+
					"(SELECT id FROM "+rule.table+" WHERE "+rule.timeColumn+" < ? LIMIT 10000)",
				cutoff)
			if res.Error != nil {
				log.Printf("[cleanup] 清理 %s 失败: %v", rule.table, res.Error)
				break
			}
			if res.RowsAffected < 10000 {
				break
			}
		}
	}
	log.Println("[cleanup] 历史数据清理完成")
}
