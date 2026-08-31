package api

import (
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"monitor/web"
)

// mountStatic 挂载内嵌前端资源：/assets 长缓存、其余路径 SPA 兜底返回 index.html。
// 若前端未构建（dist 为空），则跳过，页面由 Vite dev server 提供。
func (a *API) mountStatic(r *gin.Engine) {
	dist, err := fs.Sub(webassets.FS, "dist")
	if err != nil {
		log.Printf("[static] 读取内嵌前端资源失败: %v", err)
		return
	}
	if _, err := dist.Open("index.html"); err != nil {
		log.Println("[static] 未检测到前端构建产物（index.html 缺失），跳过静态服务（开发模式）")
		return
	}
	fileServer := http.FileServer(http.FS(dist))

	r.GET("/assets/*filepath", func(c *gin.Context) {
		// Vite 产物带 hash，可长缓存
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// API 未知路径返回 JSON 404
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "接口不存在"})
			return
		}
		// 真实存在的静态文件（favicon 等）直接返回
		rel := strings.TrimPrefix(path, "/")
		if rel != "" {
			if f, err := dist.Open(rel); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		// SPA 兜底：返回 index.html
		c.Header("Cache-Control", "no-cache")
		index, err := dist.Open("index.html")
		if err != nil {
			c.String(http.StatusNotFound, "not found")
			return
		}
		defer index.Close()
		data, _ := io.ReadAll(index)
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}
