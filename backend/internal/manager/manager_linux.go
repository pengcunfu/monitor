//go:build linux

package manager

// Linux：进程管理用 unix 实现，服务管理用 systemd D-Bus。
func newProcessManager() ProcessManager { return &unixProcessManager{} }

func newServiceManager() ServiceManager { return &systemdServiceManager{} }

func serviceManageSupported() bool { return true }

func platformName() string { return "linux" }
