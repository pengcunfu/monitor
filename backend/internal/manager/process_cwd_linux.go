//go:build linux

package manager

import (
	"context"
	"fmt"
	"os"
)

// processCwd Linux 读 /proc/<pid>/cwd 符号链接，失败返回空串。
func processCwd(_ context.Context, pid int32) string {
	dest, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	if err != nil {
		return ""
	}
	return dest
}
