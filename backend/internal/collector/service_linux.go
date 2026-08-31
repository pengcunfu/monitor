//go:build linux

package collector

import (
	"log"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"

	"monitor/internal/model"
)

// collectServices 采集 systemd 服务状态并落库、广播。
// 普通用户通常即可访问 systemd system bus。
func (c *Collector) collectServices() {
	conn, err := dbus.NewSystemConnection()
	if err != nil {
		log.Printf("[collector] 连接 systemd D-Bus 失败，服务采集不可用: %v", err)
		return
	}
	defer conn.Close()

	units, err := conn.ListUnits()
	if err != nil {
		log.Printf("[collector] 获取 systemd 单元列表失败: %v", err)
		return
	}

	now := time.Now().UnixMilli()
	var states []model.ServiceState
	for _, u := range units {
		if !strings.HasSuffix(u.Name, ".service") {
			continue
		}
		states = append(states, model.ServiceState{
			Ts:          now,
			Name:        u.Name,
			Description: u.Description,
			LoadState:   u.LoadState,
			ActiveState: u.ActiveState,
			SubState:    u.SubState,
			IsActive:    u.ActiveState == "active",
		})
	}
	if len(states) == 0 {
		return
	}
	if err := c.st.InsertServiceStates(states); err != nil {
		log.Printf("[collector] 写入服务状态失败: %v", err)
		return
	}
	if c.updater != nil {
		c.updater.UpdateServiceStates(states)
	}
	if c.hub != nil {
		c.hub.Broadcast("service", states)
	}
}
