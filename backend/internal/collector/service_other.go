//go:build !linux

package collector

// collectServices 非 Linux 平台不采集 systemd 服务状态。
func (c *Collector) collectServices() {}
