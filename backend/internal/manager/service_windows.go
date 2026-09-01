//go:build windows

package manager

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// windowsServiceManager Windows SCM 服务管理实现（用原生 SCM API，不用 sc.exe）。
type windowsServiceManager struct{}

// connect 打开服务控制管理器；仅需 CONNECT 权限，具体操作权限在 open 时按需申请。
// 权限不足返回 ErrPermission。
func connect() (*mgr.Mgr, error) {
	h, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return nil, fmt.Errorf("%w: 请以管理员身份运行以管理服务", ErrPermission)
		}
		return nil, fmt.Errorf("%w: 连接服务控制管理器失败: %v", ErrInternal, err)
	}
	return &mgr.Mgr{Handle: h}, nil
}

// open 按 access 打开指定服务（本次操作所需的最小权限）；不存在返回 ErrNotFound。
// mgr.Mgr.OpenService 硬编码 SERVICE_ALL_ACCESS（需要管理员），故用底层 API 自行打开。
func open(m *mgr.Mgr, name string, access uint32) (*mgr.Service, error) {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, fmt.Errorf("%w: 服务名非法: %v", ErrInternal, err)
	}
	h, err := windows.OpenService(m.Handle, namePtr, access)
	if err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			return nil, fmt.Errorf("%w: 服务不存在: %s", ErrNotFound, name)
		}
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return nil, fmt.Errorf("%w: 无权访问服务 %s", ErrPermission, name)
		}
		return nil, fmt.Errorf("%w: 打开服务失败: %v", ErrInternal, err)
	}
	return &mgr.Service{Name: name, Handle: h}, nil
}

// waitState 轮询服务状态直到达到目标状态（StartPending→Running 等），30s 超时。
func waitState(ctx context.Context, s *mgr.Service, want svc.State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		st, err := s.Query()
		if err != nil {
			return fmt.Errorf("%w: 查询服务状态失败: %v", ErrInternal, err)
		}
		if st.State == want {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: 请求已取消", ErrInternal)
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("%w: 等待服务状态 %v 超时", ErrInternal, want)
}

// Start 启动服务（幂等：已在运行则直接返回）。
func (m *windowsServiceManager) Start(ctx context.Context, name string) error {
	conn, err := connect()
	if err != nil {
		return err
	}
	defer conn.Disconnect()

	s, err := open(conn, name, windows.SERVICE_QUERY_STATUS|windows.SERVICE_START)
	if err != nil {
		return err
	}
	defer s.Close()

	st, err := s.Query()
	if err != nil {
		return fmt.Errorf("%w: 查询服务状态失败: %v", ErrInternal, err)
	}
	if st.State == svc.Running {
		return nil
	}
	if err := s.Start(); err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_DISABLED) {
			return fmt.Errorf("%w: 服务 %s 已被禁用，请先开启自启或手动启用", ErrConflict, name)
		}
		return fmt.Errorf("%w: 启动服务失败: %v", ErrInternal, err)
	}
	return waitState(ctx, s, svc.Running, 30*time.Second)
}

// Stop 停止服务。
func (m *windowsServiceManager) Stop(ctx context.Context, name string) error {
	conn, err := connect()
	if err != nil {
		return err
	}
	defer conn.Disconnect()

	s, err := open(conn, name, windows.SERVICE_QUERY_STATUS|windows.SERVICE_STOP)
	if err != nil {
		return err
	}
	defer s.Close()

	if _, err := s.Control(svc.Stop); err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) {
			return nil // 已停止，幂等
		}
		return fmt.Errorf("%w: 停止服务失败: %v", ErrInternal, err)
	}
	return waitState(ctx, s, svc.Stopped, 30*time.Second)
}

// Restart 重启服务：先停止等 Stopped，再启动等 Running。
func (m *windowsServiceManager) Restart(ctx context.Context, name string) error {
	conn, err := connect()
	if err != nil {
		return err
	}
	defer conn.Disconnect()

	s, err := open(conn, name, windows.SERVICE_QUERY_STATUS|windows.SERVICE_START|windows.SERVICE_STOP)
	if err != nil {
		return err
	}
	defer s.Close()

	if _, err := s.Control(svc.Stop); err != nil && !errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) {
		return fmt.Errorf("%w: 停止服务失败: %v", ErrInternal, err)
	}
	if err := waitState(ctx, s, svc.Stopped, 30*time.Second); err != nil {
		return err
	}
	if err := s.Start(); err != nil {
		return fmt.Errorf("%w: 重启服务失败: %v", ErrInternal, err)
	}
	return waitState(ctx, s, svc.Running, 30*time.Second)
}

// Enable 设置开机自启（StartType=Automatic）。
func (m *windowsServiceManager) Enable(ctx context.Context, name string) error {
	return m.setStartType(ctx, name, windows.SERVICE_AUTO_START)
}

// Disable 取消开机自启（StartType=Manual，语义对齐 systemd disable，仍可手动启动）。
func (m *windowsServiceManager) Disable(ctx context.Context, name string) error {
	return m.setStartType(ctx, name, windows.SERVICE_DEMAND_START)
}

// setStartType 修改服务启动类型。
func (m *windowsServiceManager) setStartType(_ context.Context, name string, startType uint32) error {
	conn, err := connect()
	if err != nil {
		return err
	}
	defer conn.Disconnect()

	s, err := open(conn, name, windows.SERVICE_QUERY_CONFIG|windows.SERVICE_CHANGE_CONFIG)
	if err != nil {
		return err
	}
	defer s.Close()

	cfg, err := s.Config()
	if err != nil {
		return fmt.Errorf("%w: 读取服务配置失败: %v", ErrInternal, err)
	}
	cfg.StartType = startType
	if err := s.UpdateConfig(cfg); err != nil {
		return fmt.Errorf("%w: 修改服务启动类型失败: %v", ErrInternal, err)
	}
	return nil
}
