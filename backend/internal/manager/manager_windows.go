//go:build windows

package manager

// Windows：进程管理用 windows 实现（TerminateProcess），服务管理用 SCM。
func newProcessManager() ProcessManager { return &windowsProcessManager{} }

func newServiceManager() ServiceManager { return &windowsServiceManager{} }

func serviceManageSupported() bool { return true }

func platformName() string { return "windows" }
