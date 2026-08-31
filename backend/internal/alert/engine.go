package alert

import (
	"fmt"
	"log"
	"sync"
	"time"

	"monitor/internal/model"
	"monitor/internal/store"
)

// Broadcaster 实时广播接口（由 ws.Hub 实现）。
type Broadcaster interface {
	Broadcast(topic string, data interface{})
}

// Notifier 告警通知接口（由 notifier.Manager 实现，P6 接入）。
type Notifier interface {
	SendAlert(ev *model.AlertEvent, rule *model.AlertRule, phase string)
}

// ruleState 单条「规则×目标实例」的运行时状态。
type ruleState struct {
	count      int    // 连续满足的采集周期数
	status     string // normal / firing
	eventID    uint   // 当前 firing 事件 ID（去重核心）
	lastNotify int64  // 上次通知时间（毫秒）
}

// metricValue 规则提取出的一个取值（target 用于多实例规则）。
type metricValue struct {
	target string
	value  float64
}

// Engine 告警引擎：每个采集周期评估启用中的规则，驱动状态机并落库/通知/广播。
type Engine struct {
	st       *store.Store
	hub      Broadcaster
	notifier Notifier

	mu     sync.Mutex
	rules  []*model.AlertRule
	states map[string]*ruleState // key: "<ruleID>:<target>"
	procs  []model.ProcessSample
	svcs   []model.ServiceState
}

// New 创建引擎并加载规则。
func New(st *store.Store) *Engine {
	e := &Engine{
		st:     st,
		states: map[string]*ruleState{},
	}
	e.Reload()
	return e
}

// SetHub 注入广播。
func (e *Engine) SetHub(h Broadcaster) { e.hub = h }

// SetNotifier 注入通知器。
func (e *Engine) SetNotifier(n Notifier) { e.notifier = n }

// Reload 从数据库重新加载启用中的规则（规则 CRUD 后调用；也支持 30s 自动刷新）。
func (e *Engine) Reload() {
	rules, err := e.st.ListEnabledRules()
	if err != nil {
		log.Printf("[alert] 加载规则失败: %v", err)
		return
	}
	rs := make([]*model.AlertRule, 0, len(rules))
	for i := range rules {
		rs = append(rs, &rules[i])
	}
	e.mu.Lock()
	e.rules = rs
	e.mu.Unlock()
}

// AutoReload 后台定时刷新规则缓存。
func (e *Engine) AutoReload(ctxDone <-chan struct{}) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctxDone:
			return
		case <-t.C:
			e.Reload()
		}
	}
}

// FiringCount 当前触发中的告警数量。
func (e *Engine) FiringCount() int {
	n, err := e.st.FiringCount()
	if err != nil {
		return 0
	}
	return int(n)
}

// FiringEvents 当前触发中的告警事件（最新 N 条）。
func (e *Engine) FiringEvents(limit int) []interface{} {
	evs, err := e.st.FiringEvents(limit)
	if err != nil {
		return nil
	}
	out := make([]interface{}, 0, len(evs))
	for i := range evs {
		out = append(out, &evs[i])
	}
	return out
}

// UpdateProcessSamples 采集器每次进程采样后调用。
func (e *Engine) UpdateProcessSamples(samples []model.ProcessSample) {
	e.mu.Lock()
	e.procs = samples
	e.mu.Unlock()
}

// UpdateServiceStates 采集器每次服务状态采集后调用，并立即评估服务类规则。
func (e *Engine) UpdateServiceStates(states []model.ServiceState) {
	e.mu.Lock()
	e.svcs = states
	e.mu.Unlock()
	e.evaluateServiceRules(states)
}

// Evaluate 对主指标快照评估所有规则。
func (e *Engine) Evaluate(snap *model.MetricSnapshot) {
	e.mu.Lock()
	rules := e.rules
	e.mu.Unlock()

	now := time.Now().UnixMilli()
	for _, r := range rules {
		for _, mv := range e.extractValues(r, snap) {
			ok := compare(mv.value, r.Operator, r.Threshold)
			st := e.state(r.ID, mv.target)
			e.step(r, st, mv, ok, now)
		}
	}
}

// evaluateServiceRules 对服务状态评估 service_active 类规则。
func (e *Engine) evaluateServiceRules(states []model.ServiceState) {
	e.mu.Lock()
	rules := e.rules
	e.mu.Unlock()

	now := time.Now().UnixMilli()
	statusByService := map[string]float64{}
	for _, s := range states {
		v := 0.0
		if s.IsActive {
			v = 1.0
		}
		statusByService[s.Name] = v
	}
	for _, r := range rules {
		if r.Metric != model.MetricServiceActive {
			continue
		}
		// 按 target 匹配；target 为空则遍历全部服务
		if r.Target != "" {
			v, ok := statusByService[r.Target]
			if !ok {
				continue // 服务不存在，忽略
			}
			e.step(r, e.state(r.ID, r.Target), metricValue{target: r.Target, value: v}, compare(v, r.Operator, r.Threshold), now)
			continue
		}
		for name, v := range statusByService {
			e.step(r, e.state(r.ID, name), metricValue{target: name, value: v}, compare(v, r.Operator, r.Threshold), now)
		}
	}
}

// step 状态机单步推进：NORMAL → FIRING → RESOLVED。
func (e *Engine) step(r *model.AlertRule, st *ruleState, mv metricValue, ok bool, now int64) {
	if ok {
		st.count++
		if st.status == "normal" && st.count >= r.DurationTicks {
			e.fire(r, st, mv, now)
		}
		return
	}
	if st.status == "firing" {
		e.resolve(r, st, now)
	}
	st.count = 0
	st.status = "normal"
}

// fire 触发告警：落库、通知（带冷却）、广播。
func (e *Engine) fire(r *model.AlertRule, st *ruleState, mv metricValue, now int64) {
	ev := &model.AlertEvent{
		RuleID:    r.ID,
		RuleName:  r.Name,
		Metric:    r.Metric,
		Target:    mv.target,
		Severity:  r.Severity,
		Status:    model.EventFiring,
		Message:   buildMessage(r, mv, "触发"),
		Value:     mv.value,
		Threshold: r.Threshold,
		FiredAt:   now,
	}
	if err := e.st.CreateAlertEvent(ev); err != nil {
		log.Printf("[alert] 写入告警事件失败: %v", err)
		return
	}
	st.status = "firing"
	st.eventID = ev.ID

	if len(r.ChannelIDs()) > 0 {
		cooldown := int64(r.CooldownSec) * 1000
		if cooldown <= 0 {
			cooldown = 900_000
		}
		if st.lastNotify == 0 || now-st.lastNotify >= cooldown {
			st.lastNotify = now
			e.notify(ev, r, "触发")
		}
	}
	if e.hub != nil {
		e.hub.Broadcast("alert", ev)
	}
	log.Printf("[alert] 告警触发 rule=%s target=%s value=%.2f", r.Name, mv.target, mv.value)
}

// resolve 恢复告警：更新事件为 resolved，可选恢复通知。
func (e *Engine) resolve(r *model.AlertRule, st *ruleState, now int64) {
	if st.eventID == 0 {
		return
	}
	ev, err := e.st.GetAlertEvent(st.eventID)
	if err == nil && ev != nil {
		ev.Status = model.EventResolved
		ev.ResolvedAt = now
		ev.DurationSec = (now - ev.FiredAt) / 1000
		if err := e.st.UpdateAlertEvent(ev); err != nil {
			log.Printf("[alert] 更新告警事件失败: %v", err)
		} else if r.NotifyOnResolve && len(r.ChannelIDs()) > 0 {
			e.notify(ev, r, "恢复")
		}
		if e.hub != nil {
			e.hub.Broadcast("alert", ev)
		}
		log.Printf("[alert] 告警恢复 rule=%s target=%s 持续 %ds", r.Name, ev.Target, ev.DurationSec)
	}
	st.eventID = 0
}

// notify 发送告警通知（异步，不影响状态机）。
func (e *Engine) notify(ev *model.AlertEvent, r *model.AlertRule, phase string) {
	if e.notifier == nil {
		return
	}
	go func() {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("[alert] 通知协程 panic: %v", p)
			}
		}()
		e.notifier.SendAlert(ev, r, phase)
	}()
}

// state 获取（或创建）某规则×目标的状态。
func (e *Engine) state(ruleID uint, target string) *ruleState {
	key := fmt.Sprintf("%d:%s", ruleID, target)
	e.mu.Lock()
	defer e.mu.Unlock()
	if st, ok := e.states[key]; ok {
		return st
	}
	st := &ruleState{status: "normal"}
	e.states[key] = st
	return st
}

// compare 阈值比较。
func compare(value float64, op string, threshold float64) bool {
	switch op {
	case model.OpGT:
		return value > threshold
	case model.OpGE:
		return value >= threshold
	case model.OpLT:
		return value < threshold
	case model.OpLE:
		return value <= threshold
	default:
		return false
	}
}

// buildMessage 生成告警描述。
func buildMessage(r *model.AlertRule, mv metricValue, phase string) string {
	label := MetricLabel(r.Metric)
	if mv.target != "" {
		label = fmt.Sprintf("%s（%s）", label, mv.target)
	}
	opName := map[string]string{model.OpGT: ">", model.OpGE: ">=", model.OpLT: "<", model.OpLE: "<="}[r.Operator]
	return fmt.Sprintf("%s：%s 当前值 %.2f %s 阈值 %.2f（连续 %d 个周期）",
		phase, label, mv.value, opName, r.Threshold, r.DurationTicks)
}
