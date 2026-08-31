package alert

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"monitor/internal/model"
	"monitor/internal/store"
)

// mockNotifier 记录通知调用次数。
type mockNotifier struct {
	mu                 sync.Mutex
	fired, resolved    int
	lastFiredAt        time.Time
}

func (m *mockNotifier) SendAlert(ev *model.AlertEvent, r *model.AlertRule, phase string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if phase == "触发" {
		m.fired++
		m.lastFiredAt = time.Now()
	} else {
		m.resolved++
	}
}

func (m *mockNotifier) counts() (int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fired, m.resolved
}

func newTestEngine(t *testing.T) (*Engine, *mockNotifier, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("打开测试库失败: %v", err)
	}
	e := New(st)
	mn := &mockNotifier{}
	e.SetNotifier(mn)
	t.Cleanup(func() { _ = st.Close() })
	return e, mn, st
}

func addRule(t *testing.T, st *store.Store, e *Engine, rule *model.AlertRule) {
	t.Helper()
	// 测试规则默认绑定一个假渠道，以触发通知路径
	if len(rule.ChannelIDsJSON) == 0 {
		var j model.JSON
		_ = j.Set([]uint{1})
		rule.ChannelIDsJSON = j
	}
	if err := st.CreateRule(rule); err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}
	e.Reload()
}

func snap(cpu float64) *model.MetricSnapshot {
	return &model.MetricSnapshot{CPUUsage: cpu}
}

func waitNotifier(t *testing.T, mn *mockNotifier, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		fired, _ := mn.counts()
		if fired >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	fired, _ := mn.counts()
	t.Fatalf("通知未到达: 期望 >=%d 次触发通知, 实际 %d", want, fired)
}

// TestTriggerAndResolve 验证「CPU>50% 持续 2 个周期触发，恢复后 resolved」。
func TestTriggerAndResolve(t *testing.T) {
	e, mn, st := newTestEngine(t)
	addRule(t, st, e, &model.AlertRule{
		Name: "cpu-high", Metric: model.MetricCPUUsage, Operator: model.OpGT,
		Threshold: 50, DurationTicks: 2, CooldownSec: 900,
	})

	e.Evaluate(snap(30))  // 低于阈值，不计
	e.Evaluate(snap(80))  // 第 1 个满足周期
	e.Evaluate(snap(85))  // 第 2 个满足周期 → 触发
	e.Evaluate(snap(90))  // 持续 firing，不重复落库
	e.Evaluate(snap(10))  // 恢复

	// 恢复后应无 firing
	n, _ := st.FiringCount()
	if n != 0 {
		t.Fatalf("期望 0 条 firing, 实际 %d", n)
	}
	// 应只有 1 条 resolved 事件
	_, total, _ := st.ListAlertEvents(model.EventResolved, 0, 0, 1, 10)
	if total != 1 {
		t.Fatalf("期望 1 条 resolved 事件, 实际 %d", total)
	}
	// 触发通知应恰好 1 次
	waitNotifier(t, mn, 1)
	fired, _ := mn.counts()
	if fired != 1 {
		t.Fatalf("触发通知期望 1 次, 实际 %d", fired)
	}
}

// TestDurationNotEnough 验证未达到持续周期数不触发。
func TestDurationNotEnough(t *testing.T) {
	e, mn, st := newTestEngine(t)
	addRule(t, st, e, &model.AlertRule{
		Name: "cpu-flap", Metric: model.MetricCPUUsage, Operator: model.OpGT,
		Threshold: 50, DurationTicks: 3, CooldownSec: 900,
	})
	e.Evaluate(snap(80))
	e.Evaluate(snap(10)) // 中断
	e.Evaluate(snap(80))
	e.Evaluate(snap(80)) // 累计 2 个周期但未连续 3 个

	n, _ := st.FiringCount()
	if n != 0 {
		t.Fatalf("期望 0 条 firing, 实际 %d", n)
	}
	fired, _ := mn.counts()
	if fired != 0 {
		t.Fatalf("不应有通知, 实际 %d", fired)
	}
}

// TestCooldown 验证冷却期内再次触发不重复通知。
func TestCooldown(t *testing.T) {
	e, mn, st := newTestEngine(t)
	addRule(t, st, e, &model.AlertRule{
		Name: "cooldown-rule", Metric: model.MetricCPUUsage, Operator: model.OpGT,
		Threshold: 50, DurationTicks: 1, CooldownSec: 60,
	})

	e.Evaluate(snap(80)) // 触发
	waitNotifier(t, mn, 1)
	e.Evaluate(snap(10)) // 恢复
	e.Evaluate(snap(85)) // 冷却期内再次触发：产生新事件但不通知

	time.Sleep(100 * time.Millisecond)
	fired, _ := mn.counts()
	if fired != 1 {
		t.Fatalf("冷却期内不应重复通知, 实际 %d", fired)
	}
	// 确实产生了两条事件（一次触发一次恢复再触发）
	n, _ := st.FiringCount()
	if n != 1 {
		t.Fatalf("期望 1 条 firing, 实际 %d", n)
	}
}

// TestResolveNotify 验证 notify_on_resolve 生效。
func TestResolveNotify(t *testing.T) {
	e, mn, st := newTestEngine(t)
	addRule(t, st, e, &model.AlertRule{
		Name: "resolve-notify", Metric: model.MetricMemUsage, Operator: model.OpGE,
		Threshold: 90, DurationTicks: 1, CooldownSec: 60, NotifyOnResolve: true,
	})
	// 内存使用率用 mem_usage 字段（快照直接构造）
	e.Evaluate(&model.MetricSnapshot{MemUsage: 95})
	waitNotifier(t, mn, 1)
	e.Evaluate(&model.MetricSnapshot{MemUsage: 50}) // 恢复

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		f, r := mn.counts()
		if r >= 1 {
			_ = f
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	fired, resolved := mn.counts()
	t.Fatalf("恢复通知未发送: fired=%d resolved=%d", fired, resolved)
}
