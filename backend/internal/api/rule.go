package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"monitor/internal/model"
)

// ruleReq 规则创建/更新请求。
type ruleReq struct {
	Name            string  `json:"name"`
	Metric          string  `json:"metric"`
	Target          string  `json:"target"`
	Operator        string  `json:"operator"`
	Threshold       float64 `json:"threshold"`
	DurationTicks   int     `json:"duration_ticks"`
	Severity        string  `json:"severity"`
	ChannelIDs      []uint  `json:"channel_ids"`
	CooldownSec     int     `json:"cooldown_sec"`
	NotifyOnResolve bool    `json:"notify_on_resolve"`
	Enabled         bool    `json:"enabled"`
	Description     string  `json:"description"`
}

// ruleValidate 校验规则字段。
func ruleValidate(req *ruleReq) string {
	if req.Name == "" {
		return "请填写规则名称"
	}
	switch req.Metric {
	case model.MetricCPUUsage, model.MetricMemUsage, model.MetricLoad1,
		model.MetricDiskUsedPct, model.MetricNetRXBps, model.MetricNetTXBps,
		model.MetricServiceActive, model.MetricProcessCPU:
	default:
		return "不支持的指标类型"
	}
	switch req.Operator {
	case model.OpGT, model.OpGE, model.OpLT, model.OpLE:
	default:
		return "不支持的操作符"
	}
	if req.DurationTicks < 1 {
		req.DurationTicks = 1
	}
	if req.Severity != model.SeverityCritical {
		req.Severity = model.SeverityWarning
	}
	if req.CooldownSec <= 0 {
		req.CooldownSec = 900
	}
	if req.ChannelIDs == nil {
		req.ChannelIDs = []uint{}
	}
	return ""
}

func (req *ruleReq) toModel() (*model.AlertRule, error) {
	rule := &model.AlertRule{
		Name:            req.Name,
		Metric:          req.Metric,
		Target:          req.Target,
		Operator:        req.Operator,
		Threshold:       req.Threshold,
		DurationTicks:   req.DurationTicks,
		Severity:        req.Severity,
		CooldownSec:     req.CooldownSec,
		NotifyOnResolve: req.NotifyOnResolve,
		Enabled:         req.Enabled,
		Description:     req.Description,
	}
	if err := rule.ChannelIDsJSON.Set(req.ChannelIDs); err != nil {
		return nil, err
	}
	return rule, nil
}

// ruleList 分页查询规则。
func (a *API) ruleList(c *gin.Context) {
	page := atoiDefault(c.Query("page"), 1)
	size := atoiDefault(c.Query("size"), 20)
	if size > 100 {
		size = 100
	}
	list, total, err := a.store.ListRules(page, size)
	if err != nil {
		internalError(c, "查询规则失败")
		return
	}
	okPage(c, list, total, page, size)
}

// ruleCreate 新建规则。
func (a *API) ruleCreate(c *gin.Context) {
	var req ruleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "参数格式错误")
		return
	}
	if msg := ruleValidate(&req); msg != "" {
		badRequest(c, msg)
		return
	}
	rule, err := req.toModel()
	if err != nil {
		badRequest(c, "渠道参数错误")
		return
	}
	if err := a.store.CreateRule(rule); err != nil {
		internalError(c, "创建规则失败")
		return
	}
	a.alertEngine.Reload()
	ok(c, rule)
}

// ruleGet 规则详情。
func (a *API) ruleGet(c *gin.Context) {
	rule, err := a.store.GetRule(idParam(c))
	if err != nil || rule == nil {
		notFound(c, "规则不存在")
		return
	}
	ok(c, rule)
}

// ruleUpdate 更新规则。
func (a *API) ruleUpdate(c *gin.Context) {
	rule, err := a.store.GetRule(idParam(c))
	if err != nil || rule == nil {
		notFound(c, "规则不存在")
		return
	}
	var req ruleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "参数格式错误")
		return
	}
	if msg := ruleValidate(&req); msg != "" {
		badRequest(c, msg)
		return
	}
	upd, err := req.toModel()
	if err != nil {
		badRequest(c, "渠道参数错误")
		return
	}
	upd.ID = rule.ID
	upd.CreatedAt = rule.CreatedAt
	if err := a.store.UpdateRule(upd); err != nil {
		internalError(c, "更新规则失败")
		return
	}
	a.alertEngine.Reload()
	ok(c, upd)
}

// ruleDelete 删除规则。
func (a *API) ruleDelete(c *gin.Context) {
	if err := a.store.DeleteRule(idParam(c)); err != nil {
		internalError(c, "删除规则失败")
		return
	}
	a.alertEngine.Reload()
	ok(c, nil)
}

// ruleToggle 启用/停用规则。
func (a *API) ruleToggle(c *gin.Context) {
	enabled, _ := strconv.ParseBool(c.DefaultQuery("enabled", "true"))
	if err := a.store.ToggleRule(idParam(c), enabled); err != nil {
		internalError(c, "切换规则状态失败")
		return
	}
	a.alertEngine.Reload()
	ok(c, gin.H{"enabled": enabled})
}

// ruleReload 立即重载规则缓存。
func (a *API) ruleReload(c *gin.Context) {
	a.alertEngine.Reload()
	ok(c, gin.H{"reloaded": true})
}

// alertFiringCount 当前 firing 告警数（总览用）。
func (a *API) alertFiringCount(c *gin.Context) {
	ok(c, gin.H{"firing": a.alertEngine.FiringCount()})
}

// alertFiringList 当前 firing 告警列表。
func (a *API) alertFiringList(c *gin.Context) {
	limit := atoiDefault(c.Query("limit"), 20)
	ok(c, a.alertEngine.FiringEvents(limit))
}
