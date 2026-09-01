//go:build !linux

package manager

import "context"

// processCwd 非 Linux 平台无可靠进程工作目录 API，返回空串（启动时继承 monitor 工作目录）。
func processCwd(_ context.Context, _ int32) string { return "" }
