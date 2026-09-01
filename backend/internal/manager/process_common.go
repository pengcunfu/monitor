package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// processLaunchInfo 进程重启所需的启动信息。
type processLaunchInfo struct {
	Exe  string
	Args []string
	Cwd  string
	Name string
}

// resolveLaunchInfo 解析存活进程的启动信息（gopsutil）。
// 内核线程/已退出进程 Exe 为空，无法重启，返回 ErrUnsupported。
func resolveLaunchInfo(ctx context.Context, pid int32) (*processLaunchInfo, error) {
	p, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("%w: 进程不存在或已退出 (pid=%d)", ErrNotFound, pid)
	}

	info := &processLaunchInfo{}
	if v, err := p.ExeWithContext(ctx); err == nil && v != "" {
		info.Exe = v
	} else {
		return nil, fmt.Errorf("%w: 无法获取可执行路径，系统进程或内核线程无法重启 (pid=%d)", ErrUnsupported, pid)
	}
	if v, err := p.CmdlineSliceWithContext(ctx); err == nil {
		info.Args = v
	}
	if v, err := p.NameWithContext(ctx); err == nil {
		info.Name = v
	}
	if info.Name == "" {
		info.Name = fmt.Sprintf("pid-%d", pid)
	}
	// 工作目录尽力获取（processCwd 由各平台文件实现），失败不阻断重启
	info.Cwd = processCwd(ctx, pid)
	return info, nil
}

// openRestartLog 为被重启进程打开输出日志，便于排障（写 /tmp 而非 /dev/null）。
func openRestartLog(pid int32) (*os.File, error) {
	name := filepath.Join(os.TempDir(), fmt.Sprintf("monitor-restart-%d-%d.log", pid, time.Now().Unix()))
	return os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}
