//go:build windows

package manager

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/sys/windows"
)

// windowsProcessManager Windows 进程管理：TerminateProcess + DETACHED_PROCESS 重启。
type windowsProcessManager struct{}

// Kill 结束进程：Windows 无优雅信号，Terminate 即 TerminateProcess，轮询确认退出。
func (m *windowsProcessManager) Kill(ctx context.Context, pid int32) (*ProcessOpResult, error) {
	p, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("%w: 进程不存在或已退出 (pid=%d)", ErrNotFound, pid)
	}
	name := procName(ctx, p)

	if err := p.TerminateWithContext(ctx); err != nil {
		// 权限不足/进程已退出/受保护系统进程
		running, _ := p.IsRunningWithContext(ctx)
		if !running {
			return &ProcessOpResult{PID: pid, Name: name, Action: "killed"}, nil
		}
		if kerr := p.KillWithContext(ctx); kerr != nil {
			return nil, mapKillError(kerr, pid)
		}
	}
	if waitExited(ctx, p, 5*time.Second) {
		return &ProcessOpResult{PID: pid, Name: name, Action: "killed"}, nil
	}
	if err := p.KillWithContext(ctx); err != nil {
		return nil, mapKillError(err, pid)
	}
	return &ProcessOpResult{PID: pid, Name: name, Action: "killed"}, nil
}

// Restart 重启进程：解析启动信息 → 结束旧进程 → 以新进程组脱离拉起。
func (m *windowsProcessManager) Restart(ctx context.Context, pid int32) (*ProcessOpResult, error) {
	info, err := resolveLaunchInfo(ctx, pid)
	if err != nil {
		return nil, err
	}
	if _, err := m.Kill(ctx, pid); err != nil {
		return nil, err
	}
	newPID, err := spawnDetached(ctx, pid, info)
	if err != nil {
		return nil, err
	}
	return &ProcessOpResult{PID: pid, Name: info.Name, Exe: info.Exe, NewPID: newPID, Action: "restarted"}, nil
}

// spawnDetached 以独立进程组 + 脱离窗口 + 隐藏窗口拉起进程。
func spawnDetached(ctx context.Context, pid int32, info *processLaunchInfo) (int32, error) {
	cmd := exec.CommandContext(ctx, info.Exe, info.Args...)
	cmd.Dir = info.Cwd
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
		HideWindow:    true,
	}
	lf, err := openRestartLog(pid)
	if err == nil {
		cmd.Stdout = lf
		cmd.Stderr = lf
	}
	if err := cmd.Start(); err != nil {
		return 0, mapSpawnError(err)
	}
	go cmd.Wait()
	return int32(cmd.Process.Pid), nil
}

// waitExited 轮询进程是否已退出，最多等 timeout。
func waitExited(ctx context.Context, p *process.Process, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, err := p.IsRunningWithContext(ctx)
		if err != nil || !running {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(200 * time.Millisecond):
		}
	}
	return false
}

// procName 尽力取进程名。
func procName(ctx context.Context, p *process.Process) string {
	if v, err := p.NameWithContext(ctx); err == nil && v != "" {
		return v
	}
	return fmt.Sprintf("pid-%d", p.Pid)
}

// mapKillError 归类终止进程错误（Windows 权限错误）。
func mapKillError(err error, pid int32) error {
	if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return fmt.Errorf("%w: 无权终止进程，请以管理员身份运行 (pid=%d)", ErrPermission, pid)
	}
	return fmt.Errorf("%w: 终止进程失败 (pid=%d): %v", ErrInternal, pid, err)
}

// mapSpawnError 归类拉起进程错误。
func mapSpawnError(err error) error {
	if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return fmt.Errorf("%w: 无权启动进程，请以管理员身份运行", ErrPermission)
	}
	return fmt.Errorf("%w: 启动进程失败: %v", ErrInternal, err)
}
