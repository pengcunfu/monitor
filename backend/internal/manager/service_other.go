//go:build !linux && !windows

package manager

import (
	"context"
	"fmt"
	"runtime"
)

// unsupportedServiceManager 非 Linux/Windows 平台（如 macOS）暂不支持服务管理，
// 预留：后续新增 service_darwin.go（launchctl bootstrap/bootout/print）并把
// 本文件 build tag 收窄为 !linux && !windows && !darwin。
type unsupportedServiceManager struct{}

func (m *unsupportedServiceManager) unsupported() error {
	return fmt.Errorf("%w: %s 平台暂不支持服务管理", ErrUnsupported, runtime.GOOS)
}

func (m *unsupportedServiceManager) Start(context.Context, string) error  { return m.unsupported() }
func (m *unsupportedServiceManager) Stop(context.Context, string) error   { return m.unsupported() }
func (m *unsupportedServiceManager) Restart(context.Context, string) error { return m.unsupported() }
func (m *unsupportedServiceManager) Enable(context.Context, string) error { return m.unsupported() }
func (m *unsupportedServiceManager) Disable(context.Context, string) error { return m.unsupported() }
