package notifier

import (
	"context"
	"fmt"

	"monitor/internal/model"
)

type wecomConfig struct {
	WebhookURL string `json:"webhook_url"`
}

type wecomNotifier struct {
	cfg wecomConfig
}

func newWecom(c *model.NotificationChannel) (Notifier, error) {
	var cfg wecomConfig
	if err := c.ConfigJSON.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("企业微信配置解析失败: %w", err)
	}
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("企业微信 webhook_url 不能为空")
	}
	return &wecomNotifier{cfg: cfg}, nil
}

func (n *wecomNotifier) Send(ctx context.Context, msg *Message) error {
	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]interface{}{
			"content": fmt.Sprintf("%s\n%s", msg.Title, msg.Content),
		},
	}
	return postJSON(ctx, n.cfg.WebhookURL, payload, "errcode")
}
