//go:build !linux && !windows

package collector

// collectServices 非 Linux/Windows 平台（如 macOS）暂不采集系统服务状态。
func (c *Collector) collectServices() {}
