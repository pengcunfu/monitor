//go:build !linux && !windows

package manager

import "runtime"

// 其他平台（如 macOS）：进程管理 gopsutil 可用，服务管理暂不支持（预留 launchd 实现）。
func newProcessManager() ProcessManager { return &unixProcessManager{} }

func newServiceManager() ServiceManager { return &unsupportedServiceManager{} }

func serviceManageSupported() bool { return false }

func platformName() string { return runtime.GOOS }
