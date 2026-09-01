package api

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"monitor/internal/model"
)

// 内置 SMTP 通知渠道的固定名称（系统设置页配置，自动同步到通知渠道供告警规则选用）。
const smtpChannelName = "邮件告警"

// smtpSetting 系统设置页的邮件 SMTP 配置（与通知渠道 SMTP config 字段对齐）。
type smtpSetting struct {
	Host               string   `json:"host"`
	Port               int      `json:"port"`
	User               string   `json:"user"`
	Password           string   `json:"password"`
	From               string   `json:"from"`
	To                 []string `json:"to"`
	TLS                bool     `json:"tls"`
	InsecureSkipVerify bool     `json:"insecure_skip_verify"`
	Enabled            bool     `json:"enabled"`
}

func (s *smtpSetting) validate() string {
	switch {
	case s.Host == "":
		return "请填写 SMTP 服务器"
	case s.Port <= 0 || s.Port > 65535:
		return "端口不合法"
	case s.From == "":
		return "请填写发件人邮箱"
	case len(s.To) == 0:
		return "请至少填写一个收件人"
	}
	return ""
}

func (s *smtpSetting) toConfig() map[string]interface{} {
	return map[string]interface{}{
		"host":                 s.Host,
		"port":                 s.Port,
		"user":                 s.User,
		"password":             s.Password,
		"from":                 s.From,
		"to":                   s.To,
		"tls":                  s.TLS,
		"insecure_skip_verify": s.InsecureSkipVerify,
	}
}

// parseSmtpSetting 从设置表单值解析 SMTP 配置；无 SMTP 字段时返回 nil。
func parseSmtpSetting(v interface{}) *smtpSetting {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	s := &smtpSetting{
		Host:               strOr(m["host"]),
		User:               strOr(m["user"]),
		Password:           strOr(m["password"]),
		From:               strOr(m["from"]),
		TLS:                boolOr(m["tls"]),
		InsecureSkipVerify: boolOr(m["insecure_skip_verify"]),
		Enabled:            boolOr(m["enabled"]),
	}
	s.Port = intOr(m["port"], 465)
	if to, ok := m["to"].([]interface{}); ok {
		for _, t := range to {
			if s2, ok := t.(string); ok && s2 != "" {
				s.To = append(s.To, s2)
			}
		}
	}
	return s
}

func strOr(v interface{}) string {
	s, _ := v.(string)
	return s
}

func boolOr(v interface{}) bool {
	b, _ := v.(bool)
	return b
}

func intOr(v interface{}, def int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return def
	}
}

// settingGet 返回全部设置（含邮件 SMTP 配置，密码脱敏）。
func (a *API) settingGet(c *gin.Context) {
	settings, err := a.store.GetSettings()
	if err != nil {
		internalError(c, "查询设置失败")
		return
	}
	settings["smtp"] = a.smtpSettingResponse()
	ok(c, settings)
}

// smtpSettingResponse 返回设置页可见的 SMTP 配置（从内置渠道读取，密码脱敏）。
func (a *API) smtpSettingResponse() *smtpSetting {
	ch, err := a.store.FindChannelByNameType(smtpChannelName, model.ChannelSMTP)
	if err != nil || ch == nil {
		return &smtpSetting{Enabled: true, TLS: true}
	}
	var cfg map[string]interface{}
	_ = ch.ConfigJSON.Unmarshal(&cfg)
	s := &smtpSetting{
		Host:               strOr(cfg["host"]),
		User:               strOr(cfg["user"]),
		Password:           "***", // 脱敏
		From:               strOr(cfg["from"]),
		TLS:                boolOr(cfg["tls"]),
		InsecureSkipVerify: boolOr(cfg["insecure_skip_verify"]),
		Enabled:            ch.Enabled,
	}
	s.Port = intOr(cfg["port"], 465)
	if to, ok := cfg["to"].([]interface{}); ok {
		for _, t := range to {
			if s2, ok := t.(string); ok && s2 != "" {
				s.To = append(s.To, s2)
			}
		}
	}
	return s
}

// settingUpdate 批量更新设置；含 smtp 字段时同步内置 SMTP 通知渠道。
func (a *API) settingUpdate(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "参数格式错误")
		return
	}
	if raw, ok := body["smtp"]; ok {
		if msg := a.saveSMTPSetting(raw); msg != "" {
			badRequest(c, msg)
			return
		}
		delete(body, "smtp")
	}
	for k, v := range body {
		if err := a.store.SetSetting(k, v); err != nil {
			internalError(c, "保存设置失败")
			return
		}
	}
	ok(c, nil)
}

// saveSMTPSetting 保存邮件 SMTP 配置：更新或创建内置 SMTP 通知渠道。
func (a *API) saveSMTPSetting(v interface{}) string {
	s := parseSmtpSetting(v)
	if s == nil {
		return "SMTP 配置格式错误"
	}
	if msg := s.validate(); msg != "" {
		return msg
	}

	old, err := a.store.FindChannelByNameType(smtpChannelName, model.ChannelSMTP)
	if err != nil {
		return "查询内置渠道失败"
	}
	cfg := s.toConfig()
	if old != nil {
		// 密码留空或 *** 时保留原值
		if s.Password == "" || s.Password == "***" {
			var oldCfg map[string]interface{}
			_ = old.ConfigJSON.Unmarshal(&oldCfg)
			if p, ok := oldCfg["password"].(string); ok {
				cfg["password"] = p
			}
		}
		old.Enabled = s.Enabled
		_ = old.ConfigJSON.Set(cfg)
		if err := a.store.UpdateChannel(old); err != nil {
			return "更新内置渠道失败"
		}
		return ""
	}
	if s.Password == "" {
		return "请填写 SMTP 密码（首次配置必填）"
	}
	ch := &model.NotificationChannel{Name: smtpChannelName, Type: model.ChannelSMTP, Enabled: s.Enabled}
	_ = ch.ConfigJSON.Set(cfg)
	if err := a.store.CreateChannel(ch); err != nil {
		return "创建内置渠道失败"
	}
	return ""
}

// settingSMTPTest 用当前表单提交的 SMTP 配置发送测试邮件（未保存也可测试）。
func (a *API) settingSMTPTest(c *gin.Context) {
	var body struct {
		SMTP smtpSetting `json:"smtp"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "参数格式错误")
		return
	}
	if msg := body.SMTP.validate(); msg != "" {
		badRequest(c, msg)
		return
	}
	if body.SMTP.Password == "" || body.SMTP.Password == "***" {
		// 测试时密码未填：尝试从已保存的渠道读取
		ch, err := a.store.FindChannelByNameType(smtpChannelName, model.ChannelSMTP)
		if err != nil || ch == nil {
			badRequest(c, "未填写 SMTP 密码")
			return
		}
		var cfg map[string]interface{}
		_ = ch.ConfigJSON.Unmarshal(&cfg)
		if p, ok := cfg["password"].(string); ok && p != "" {
			body.SMTP.Password = p
		} else {
			badRequest(c, "未填写 SMTP 密码")
			return
		}
	}
	if err := a.mgr.TestSMTPConfig(context.Background(), body.SMTP.toConfig()); err != nil {
		badRequest(c, fmt.Sprintf("测试发送失败: %v", err))
		return
	}
	ok(c, gin.H{"sent": true})
}
