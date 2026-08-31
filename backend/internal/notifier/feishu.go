package notifier

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"monitor/internal/model"
)

type feishuConfig struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret"` // 可选，加签
}

type feishuNotifier struct {
	cfg feishuConfig
}

func newFeishu(c *model.NotificationChannel) (Notifier, error) {
	var cfg feishuConfig
	if err := c.ConfigJSON.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("飞书配置解析失败: %w", err)
	}
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("飞书 webhook_url 不能为空")
	}
	return &feishuNotifier{cfg: cfg}, nil
}

func (n *feishuNotifier) Send(ctx context.Context, msg *Message) error {
	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": fmt.Sprintf("%s\n%s", msg.Title, msg.Content),
		},
	}
	if n.cfg.Secret != "" {
		ts := time.Now().Unix()
		sign := fmt.Sprintf("%d\n%s", ts, n.cfg.Secret)
		h := hmac.New(sha256.New, []byte(n.cfg.Secret))
		h.Write([]byte(sign))
		payload["timestamp"] = strconv.FormatInt(ts, 10)
		payload["sign"] = base64.StdEncoding.EncodeToString(h.Sum(nil))
	}
	return postJSON(ctx, n.cfg.WebhookURL, payload, "code")
}
