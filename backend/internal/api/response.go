package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// atoiDefault 解析正整数，空/非法返回默认值。
func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		return n
	}
	return def
}

// idParam 解析路径参数 :id 为 uint。
func idParam(c *gin.Context) uint {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(id)
}

// Response 统一 JSON 响应：code=0 表示成功。
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// PageData 分页数据。
type PageData struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Msg: "ok", Data: data})
}

func okPage(c *gin.Context, list interface{}, total int64, page, size int) {
	ok(c, PageData{List: list, Total: total, Page: page, Size: size})
}

func fail(c *gin.Context, httpStatus, code int, msg string) {
	c.JSON(httpStatus, Response{Code: code, Msg: msg})
}

func badRequest(c *gin.Context, msg string) {
	fail(c, http.StatusBadRequest, 400, msg)
}

func unauthorized(c *gin.Context, msg string) {
	fail(c, http.StatusUnauthorized, 401, msg)
}

func notFound(c *gin.Context, msg string) {
	fail(c, http.StatusNotFound, 404, msg)
}

func internalError(c *gin.Context, msg string) {
	fail(c, http.StatusInternalServerError, 500, msg)
}
