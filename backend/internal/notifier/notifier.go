package notifier

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"monitor/internal/model"
	"monitor/internal/store"
)

// Message 通知消息。
type Message struct {
	Title    string
	Content  string
	Severity string
	Time     int64 // 毫秒
}

// Notifier 单渠道发送器。
type Notifier interface {
	// Send 发送一条告警/测试消息。
	Send(ctx context.Context, msg *Message) error
}

// Manager 通知管理器：按渠道构建发送器、分发、记录发送日志。
type Manager struct {
	st *store.Store
}

// NewManager 创建通知管理器。
func NewManager(st *store.Store) *Manager {
	return &Manager{st: st}
}

// buildNotifier 根据渠道类型与配置构建发送器。
func buildNotifier(c *model.NotificationChannel) (Notifier, error) {
	switch c.Type {
	case model.ChannelSMTP:
		return newSMTP(c)
	case model.ChannelWebhook:
		return newWebhook(c)
	case model.ChannelFeishu:
		return newFeishu(c)
	case model.ChannelWecom:
		return newWecom(c)
	case model.ChannelDingTalk:
		return newDingTalk(c)
	case model.ChannelServerChan:
		return newServerChan(c)
	default:
		return nil, fmt.Errorf("未知渠道类型 %q", c.Type)
	}
}

// SendAlert 实现 alert.Notifier 接口：由告警引擎在触发/恢复时调用。
func (m *Manager) SendAlert(ev *model.AlertEvent, rule *model.AlertRule, phase string) {
	ids := rule.ChannelIDs()
	if len(ids) == 0 {
		return
	}
	msg := &Message{
		Title:    fmt.Sprintf("[监控平台][%s] %s - %s", ev.Severity, phase, ev.RuleName),
		Content:  ev.Message,
		Severity: ev.Severity,
		Time:     time.Now().UnixMilli(),
	}
	m.SendToChannels(context.Background(), ids, msg, ev.ID)
}

// SendToChannels 向指定渠道列表发送消息并逐条记录日志。
func (m *Manager) SendToChannels(ctx context.Context, ids []uint, msg *Message, eventID uint) {
	for _, id := range ids {
		ch, err := m.st.GetChannel(id)
		if err != nil || ch == nil {
			log.Printf("[notifier] 渠道 %d 不存在或查询失败", id)
			continue
		}
		n, err := buildNotifier(ch)
		if err != nil {
			m.logSend(ch, eventID, msg.Title, false, err.Error())
			continue
		}
		if err := n.Send(ctx, msg); err != nil {
			m.logSend(ch, eventID, msg.Title, false, err.Error())
			log.Printf("[notifier] 渠道 %s(%s) 发送失败: %v", ch.Name, ch.Type, err)
			continue
		}
		m.logSend(ch, eventID, msg.Title, true, "ok")
	}
}

// TestChannel 向单个渠道发送测试消息（供前端「测试」按钮调用）。
func (m *Manager) TestChannel(ctx context.Context, id uint) error {
	ch, err := m.st.GetChannel(id)
	if err != nil || ch == nil {
		return fmt.Errorf("渠道不存在")
	}
	n, err := buildNotifier(ch)
	if err != nil {
		return err
	}
	msg := &Message{
		Title:    "[监控平台] 测试通知",
		Content:  fmt.Sprintf("这是一条测试通知。\n时间：%s\n渠道：%s", time.Now().Format("2006-01-02 15:04:05"), ch.Name),
		Severity: model.SeverityWarning,
		Time:     time.Now().UnixMilli(),
	}
	if err := n.Send(ctx, msg); err != nil {
		m.logSend(ch, 0, msg.Title, false, err.Error())
		return err
	}
	m.logSend(ch, 0, msg.Title, true, "ok")
	return nil
}

// logSend 记录发送日志（response 截断 500 字符）。
func (m *Manager) logSend(ch *model.NotificationChannel, eventID uint, title string, success bool, response string) {
	if len(response) > 500 {
		response = response[:500]
	}
	_ = m.st.CreateNotificationLog(&model.NotificationLog{
		ChannelID: ch.ID,
		EventID:   eventID,
		Type:      ch.Type,
		Target:    maskTarget(ch),
		Title:     title,
		Success:   success,
		Response:  response,
	})
}

// maskTarget 生成脱敏的目标描述（收件人 / URL）。
func maskTarget(ch *model.NotificationChannel) string {
	switch ch.Type {
	case model.ChannelSMTP:
		var cfg smtpConfig
		if ch.ConfigJSON.Unmarshal(&cfg) == nil && len(cfg.To) > 0 {
			return strings.Join(cfg.To, ",")
		}
	case model.ChannelWebhook, model.ChannelFeishu, model.ChannelWecom, model.ChannelDingTalk:
		var cfg map[string]interface{}
		if ch.ConfigJSON.Unmarshal(&cfg) == nil {
			if u, ok := cfg["webhook_url"].(string); ok && u != "" {
				return maskURL(u)
			}
			if u, ok := cfg["url"].(string); ok && u != "" {
				return maskURL(u)
			}
		}
	case model.ChannelServerChan:
		return "sctapi.ftqq.com"
	}
	return ch.Name
}

// maskURL 脱敏 URL 中的查询参数（key 等敏感信息）。
func maskURL(u string) string {
	if i := strings.IndexByte(u, '?'); i > 0 {
		return u[:i] + "?***"
	}
	return u
}
