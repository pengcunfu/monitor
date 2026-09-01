//go:build linux

package manager

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
)

// systemdServiceManager systemd D-Bus 服务管理实现。
type systemdServiceManager struct{}

// unitName 补全 .service 后缀（采集到的名称已带后缀，前端传参可能不带）。
func unitName(name string) string {
	if strings.HasSuffix(name, ".service") {
		return name
	}
	return name + ".service"
}

// job 单个启停操作的 job 回调（StartUnit/StopUnit/RestartUnit 均此签名）。
type job func(*dbus.Conn, string, string, chan<- string) (int, error)

// startStop 统一执行启停类 job 并等待结果。
func (m *systemdServiceManager) startStop(ctx context.Context, name, mode string, fn job) error {
	conn, err := dbus.NewSystemConnection()
	if err != nil {
		return fmt.Errorf("%w: 连接 systemd 失败: %v", ErrInternal, err)
	}
	defer conn.Close()

	unit := unitName(name)
	ch := make(chan string, 1)
	if _, err := fn(conn, unit, mode, ch); err != nil {
		return fmt.Errorf("%w: %s", ErrPermission, err)
	}
	return waitJob(ctx, ch)
}

// waitJob 等待 job 结果：done → nil；canceled/其他 → ErrConflict；超时 → ErrInternal。
func waitJob(ctx context.Context, ch chan string) error {
	select {
	case result, ok := <-ch:
		if !ok {
			return fmt.Errorf("%w: 未收到 systemd job 结果", ErrInternal)
		}
		if result == "done" {
			return nil
		}
		if result == "canceled" {
			return fmt.Errorf("%w: 操作被取消", ErrConflict)
		}
		return fmt.Errorf("%w: systemd job 结果: %s", ErrConflict, result)
	case <-time.After(30 * time.Second):
		return fmt.Errorf("%w: systemd job 超时", ErrInternal)
	case <-ctx.Done():
		return fmt.Errorf("%w: 请求已取消", ErrInternal)
	}
}

// Start 启动服务单元。
func (m *systemdServiceManager) Start(ctx context.Context, name string) error {
	return m.startStop(ctx, name, "replace", func(c *dbus.Conn, u, mode string, ch chan<- string) (int, error) {
		return c.StartUnitContext(ctx, u, mode, ch)
	})
}

// Stop 停止服务单元。
func (m *systemdServiceManager) Stop(ctx context.Context, name string) error {
	return m.startStop(ctx, name, "replace", func(c *dbus.Conn, u, mode string, ch chan<- string) (int, error) {
		return c.StopUnitContext(ctx, u, mode, ch)
	})
}

// Restart 重启服务单元。
func (m *systemdServiceManager) Restart(ctx context.Context, name string) error {
	return m.startStop(ctx, name, "replace", func(c *dbus.Conn, u, mode string, ch chan<- string) (int, error) {
		return c.RestartUnitContext(ctx, u, mode, ch)
	})
}

// Enable 设置开机自启；无 [Install] 段的单元无法 enable。
func (m *systemdServiceManager) Enable(ctx context.Context, name string) error {
	conn, err := dbus.NewSystemConnection()
	if err != nil {
		return fmt.Errorf("%w: 连接 systemd 失败: %v", ErrInternal, err)
	}
	defer conn.Close()

	unit := unitName(name)
	carriesInstallInfo, _, err := conn.EnableUnitFilesContext(ctx, []string{unit}, false, true)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrPermission, err)
	}
	if !carriesInstallInfo {
		return fmt.Errorf("%w: 单元 %s 无 [Install] 段，无法设置开机自启", ErrConflict, unit)
	}
	return nil
}

// Disable 取消开机自启。
func (m *systemdServiceManager) Disable(ctx context.Context, name string) error {
	conn, err := dbus.NewSystemConnection()
	if err != nil {
		return fmt.Errorf("%w: 连接 systemd 失败: %v", ErrInternal, err)
	}
	defer conn.Close()

	if _, err := conn.DisableUnitFilesContext(ctx, []string{unitName(name)}, false); err != nil {
		return fmt.Errorf("%w: %s", ErrPermission, err)
	}
	return nil
}
