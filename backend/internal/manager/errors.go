package manager

import "errors"

// 管理操作统一的哨兵错误，平台实现用 fmt.Errorf("%w: ...", ErrXxx) 包装，
// API 层通过 errors.Is 归类为对应 HTTP 状态码。
var (
	// ErrNotFound 目标不存在或已退出。
	ErrNotFound = errors.New("目标不存在或已退出")
	// ErrPermission 权限不足（非管理员 / 跨用户操作 / 系统进程）。
	ErrPermission = errors.New("权限不足")
	// ErrUnsupported 当前平台不支持该操作。
	ErrUnsupported = errors.New("当前平台不支持该操作")
	// ErrConflict 操作与当前状态冲突（如启动被禁用服务、单元无 [Install] 段）。
	ErrConflict = errors.New("操作与当前状态冲突")
	// ErrInternal 内部错误。
	ErrInternal = errors.New("内部错误")
)
