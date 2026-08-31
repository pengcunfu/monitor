package api

import (
	"github.com/gin-gonic/gin"
)

// settingGet 返回全部设置。
func (a *API) settingGet(c *gin.Context) {
	settings, err := a.store.GetSettings()
	if err != nil {
		internalError(c, "查询设置失败")
		return
	}
	ok(c, settings)
}

// settingUpdate 批量更新设置。
func (a *API) settingUpdate(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "参数格式错误")
		return
	}
	for k, v := range body {
		if err := a.store.SetSetting(k, v); err != nil {
			internalError(c, "保存设置失败")
			return
		}
	}
	ok(c, nil)
}
