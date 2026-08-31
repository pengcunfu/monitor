// Package webassets 内嵌前端构建产物（Vite 输出到 web/dist）。
// go:embed 路径相对于本包目录（backend/web），即 dist → backend/web/dist。
package webassets

import "embed"

//go:embed dist
var FS embed.FS
