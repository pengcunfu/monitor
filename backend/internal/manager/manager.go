package manager

import "context"

// ProcessManager 进程管理接口（kill/restart，跨平台实现）。
type ProcessManager interface {
	// Kill 结束指定进程。
	Kill(ctx context.Context, pid int32) (*ProcessOpResult, error)
	// Restart 重启指定进程：解析其启动信息，结束旧进程后重新拉起。
	Restart(ctx context.Context, pid int32) (*ProcessOpResult, error)
}

// ServiceManager 服务管理接口（start/stop/restart + 开机自启 enable/disable，跨平台实现）。
type ServiceManager interface {
	Start(ctx context.Context, name string) error
	Stop(ctx context.Context, name string) error
	Restart(ctx context.Context, name string) error
	Enable(ctx context.Context, name string) error
	Disable(ctx context.Context, name string) error
}

// ProcessOpResult 进程操作结果。
type ProcessOpResult struct {
	PID    int32  `json:"pid"`
	Name   string `json:"name"`
	Exe    string `json:"exe,omitempty"`
	NewPID int32  `json:"new_pid,omitempty"` // restart 后新进程 PID
	Action string `json:"action"`            // killed | restarted
}

// Capabilities 当前平台的能力声明，前端据此隐藏不支持的按钮。
type Capabilities struct {
	Platform      string `json:"platform"` // linux / windows / darwin / other
	ProcessManage bool   `json:"process_manage"`
	ServiceManage bool   `json:"service_manage"`
}

// Manager 聚合进程/服务管理能力，平台实现由 build tag 工厂注入。
type Manager struct {
	procs    ProcessManager
	svcs     ServiceManager
	platform string
	svcOK    bool
}

// New 创建平台对应的 Manager（进程管理通常各平台可用，服务管理视平台而定）。
func New() *Manager {
	return &Manager{
		procs:    newProcessManager(),
		svcs:     newServiceManager(),
		platform: platformName(),
		svcOK:    serviceManageSupported(),
	}
}

// Process 返回进程管理器。
func (m *Manager) Process() ProcessManager { return m.procs }

// Service 返回服务管理器。
func (m *Manager) Service() ServiceManager { return m.svcs }

// Capabilities 返回平台能力声明。
func (m *Manager) Capabilities() Capabilities {
	return Capabilities{
		Platform:      m.platform,
		ProcessManage: true,
		ServiceManage: m.svcOK,
	}
}
