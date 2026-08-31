package notifier

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"monitor/internal/model"
)

type dingtalkConfig struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret"` // 可选，加签
}

type dingtalkNotifier struct {
	cfg dingtalkConfig
}

func newDingTalk(c *model.NotificationChannel) (Notifier, error) {
	var cfg dingtalkConfig
	if err := c.ConfigJSON.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("钉钉配置解析失败: %w", err)
	}
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("钉钉 webhook_url 不能为空")
	}
	return &dingtalkNotifier{cfg: cfg}, nil
}

func (n *dingtalkNotifier) Send(ctx context.Context, msg *Message) error {
	target := n.cfg.WebhookURL
	if n.cfg.Secret != "" {
		ts := time.Now().UnixMilli()
		stringToSign := fmt.Sprintf("%d\n%s", ts, n.cfg.Secret)
		h := hmac.New(sha256.New, []byte(n.cfg.Secret))
		h.Write([]byte(stringToSign))
		sign := url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))
		sep := "?"
		if strings.Contains(target, "?") {
			sep = "&"
		}
		target = fmt.Sprintf("%s%stimestamp=%d&sign=%s", target, sep, ts, sign)
	}

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": fmt.Sprintf("[熔岩网络安全事件应急处置系统] %s", msg.Title),
			"text":  fmt.Sprintf("### %s\n\n%s", msg.Title, msg.Content),
		},
	}
	return postJSON(ctx, target, payload, "errcode")
}
