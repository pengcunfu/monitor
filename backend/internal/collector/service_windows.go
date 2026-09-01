//go:build windows

package collector

import (
	"log"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"monitor/internal/model"
)

// collectServices 采集 Windows 服务状态并落库、广播。
// 只请求读取所需的最小权限（枚举 SCM + 只读打开服务），普通用户即可读取大部分服务；
// 个别受保护的系统服务无权限读取时跳过，不阻塞其他采集。
func (c *Collector) collectServices() {
	m, err := scmConnectRead()
	if err != nil {
		log.Printf("[collector] 连接服务控制管理器失败，服务采集不可用: %v", err)
		return
	}
	defer m.Disconnect()

	names, err := m.ListServices()
	if err != nil {
		log.Printf("[collector] 获取 Windows 服务列表失败: %v", err)
		return
	}

	now := time.Now().UnixMilli()
	states := make([]model.ServiceState, 0, len(names))
	for _, name := range names {
		svcState, ok := readWindowsService(m, name, now)
		if ok {
			states = append(states, svcState)
		}
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

// scmConnectRead 以最小权限打开服务控制管理器：枚举服务即可，无需
// SC_MANAGER_ALL_ACCESS（那需要管理员）。mgr.Connect() 硬编码 ALL_ACCESS，故用底层 API 自行打开。
func scmConnectRead() (*mgr.Mgr, error) {
	h, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT|windows.SC_MANAGER_ENUMERATE_SERVICE)
	if err != nil {
		return nil, err
	}
	return &mgr.Mgr{Handle: h}, nil
}

// readWindowsService 读取单个 Windows 服务并归一化为 ServiceState；失败返回 ok=false。
func readWindowsService(m *mgr.Mgr, name string, ts int64) (model.ServiceState, bool) {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return model.ServiceState{}, false
	}
	// 只读权限即可查询状态与配置；mgr.Mgr.OpenService 硬编码 SERVICE_ALL_ACCESS（需要管理员）。
	h, err := windows.OpenService(m.Handle, namePtr, windows.SERVICE_QUERY_STATUS|windows.SERVICE_QUERY_CONFIG)
	if err != nil {
		return model.ServiceState{}, false
	}
	s := &mgr.Service{Name: name, Handle: h}
	defer s.Close()

	st, err := s.Query()
	if err != nil {
		return model.ServiceState{}, false
	}
	cfg, _ := s.Config() // 配置读取失败不影响状态

	active, sub := normalizeWindowsState(st.State)
	return model.ServiceState{
		Ts:          ts,
		Name:        name,
		Description: cfg.Description,
		LoadState:   windowsLoadState(cfg.StartType),
		ActiveState: active,
		SubState:    sub,
		IsActive:    active == "active",
		Enabled:     cfg.StartType == windows.SERVICE_AUTO_START,
		MainPID:     int32(st.ProcessId),
		ExitCode:    int32(st.Win32ExitCode),
	}, true
}

// normalizeWindowsState 归一化 Windows 服务状态为 ActiveState/SubState。
func normalizeWindowsState(st svc.State) (string, string) {
	switch st {
	case svc.Running:
		return "active", "running"
	case svc.StartPending:
		return "activating", "start"
	case svc.Stopped:
		return "inactive", "exited"
	case svc.StopPending:
		return "deactivating", "stop"
	case svc.Paused, svc.PausePending, svc.ContinuePending:
		return "paused", "paused"
	default:
		return "failed", ""
	}
}

// windowsLoadState 映射启动类型为 load_state。
func windowsLoadState(startType uint32) string {
	switch startType {
	case windows.SERVICE_DISABLED:
		return "disabled"
	default:
		return "loaded"
	}
}
