package api

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"monitor/internal/model"
)

// 渠道类型中文名。
var channelTypeNames = map[string]string{
	model.ChannelSMTP:       "邮件 SMTP",
	model.ChannelWebhook:    "通用 Webhook",
	model.ChannelFeishu:     "飞书机器人",
	model.ChannelWecom:      "企业微信机器人",
	model.ChannelDingTalk:   "钉钉机器人",
	model.ChannelServerChan: "Server酱",
}

// channelReq 渠道创建/更新请求。
type channelReq struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Config  map[string]interface{} `json:"config"`
	Enabled bool                   `json:"enabled"`
}

// sensitiveKeys 各渠道配置中的敏感字段（返回脱敏、更新留空不修改）。
var sensitiveKeys = map[string][]string{
	model.ChannelSMTP:       {"password"},
	model.ChannelFeishu:     {"secret"},
	model.ChannelDingTalk:   {"secret"},
	model.ChannelServerChan: {"sendkey"},
}

func channelValidate(req *channelReq) string {
	if req.Name == "" {
		return "请填写渠道名称"
	}
	if _, ok := channelTypeNames[req.Type]; !ok {
		return "不支持的渠道类型"
	}
	return ""
}

// sanitizeConfig 脱敏渠道配置（敏感字段有值则替换为 ***）。
func sanitizeConfig(typ string, cfg map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(cfg))
	for k, v := range cfg {
		out[k] = v
	}
	for _, key := range sensitiveKeys[typ] {
		if s, ok := out[key].(string); ok && s != "" {
			out[key] = "***"
		}
	}
	return out
}

// keepSecret 更新渠道时，若敏感字段为 "***" 或空，则沿用原配置中的值。
func keepSecret(typ string, newCfg, oldCfg map[string]interface{}) {
	for _, key := range sensitiveKeys[typ] {
		if v, ok := newCfg[key]; ok {
			s, isStr := v.(string)
			if isStr && (s == "" || s == "***") {
				if ov, ok := oldCfg[key]; ok {
					newCfg[key] = ov
				}
			}
		}
	}
}

func (req *channelReq) toModel() (*model.NotificationChannel, error) {
	ch := &model.NotificationChannel{
		Name:    req.Name,
		Type:    req.Type,
		Enabled: req.Enabled,
	}
	if err := ch.ConfigJSON.Set(req.Config); err != nil {
		return nil, err
	}
	return ch, nil
}

// channelList 渠道列表（配置脱敏）。
func (a *API) channelList(c *gin.Context) {
	list, err := a.store.ListChannels()
	if err != nil {
		internalError(c, "查询渠道失败")
		return
	}
	for i := range list {
		var cfg map[string]interface{}
		if list[i].ConfigJSON.Unmarshal(&cfg) == nil {
			_ = list[i].ConfigJSON.Set(sanitizeConfig(list[i].Type, cfg))
		}
	}
	ok(c, list)
}

// channelCreate 新建渠道。
func (a *API) channelCreate(c *gin.Context) {
	var req channelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "参数格式错误")
		return
	}
	if msg := channelValidate(&req); msg != "" {
		badRequest(c, msg)
		return
	}
	if req.Config == nil {
		req.Config = map[string]interface{}{}
	}
	ch, err := req.toModel()
	if err != nil {
		badRequest(c, "配置格式错误")
		return
	}
	if err := a.store.CreateChannel(ch); err != nil {
		internalError(c, "创建渠道失败")
		return
	}
	ok(c, ch)
}

// channelGet 渠道详情（脱敏）。
func (a *API) channelGet(c *gin.Context) {
	ch, err := a.store.GetChannel(idParam(c))
	if err != nil || ch == nil {
		notFound(c, "渠道不存在")
		return
	}
	var cfg map[string]interface{}
	if ch.ConfigJSON.Unmarshal(&cfg) == nil {
		_ = ch.ConfigJSON.Set(sanitizeConfig(ch.Type, cfg))
	}
	ok(c, ch)
}

// channelUpdate 更新渠道（敏感字段留空/*** 时保留原值）。
func (a *API) channelUpdate(c *gin.Context) {
	old, err := a.store.GetChannel(idParam(c))
	if err != nil || old == nil {
		notFound(c, "渠道不存在")
		return
	}
	var req channelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "参数格式错误")
		return
	}
	if msg := channelValidate(&req); msg != "" {
		badRequest(c, msg)
		return
	}
	if req.Config == nil {
		req.Config = map[string]interface{}{}
	}
	var oldCfg map[string]interface{}
	_ = old.ConfigJSON.Unmarshal(&oldCfg)
	keepSecret(req.Type, req.Config, oldCfg)

	upd, err := req.toModel()
	if err != nil {
		badRequest(c, "配置格式错误")
		return
	}
	upd.ID = old.ID
	upd.CreatedAt = old.CreatedAt
	if err := a.store.UpdateChannel(upd); err != nil {
		internalError(c, "更新渠道失败")
		return
	}
	ok(c, upd)
}

// channelDelete 删除渠道。
func (a *API) channelDelete(c *gin.Context) {
	if err := a.store.DeleteChannel(idParam(c)); err != nil {
		internalError(c, "删除渠道失败")
		return
	}
	ok(c, nil)
}

// channelTest 测试发送。
func (a *API) channelTest(c *gin.Context) {
	if err := a.mgr.TestChannel(context.Background(), idParam(c)); err != nil {
		badRequest(c, fmt.Sprintf("发送失败: %v", err))
		return
	}
	ok(c, gin.H{"sent": true})
}

// channelTypes 支持的渠道类型列表。
func (a *API) channelTypes(c *gin.Context) {
	out := make([]gin.H, 0, len(channelTypeNames))
	for k, v := range channelTypeNames {
		out = append(out, gin.H{"type": k, "name": v})
	}
	ok(c, out)
}
