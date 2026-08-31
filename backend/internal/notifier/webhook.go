package notifier

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"text/template"

	"monitor/internal/model"
)

// defaultWebhookTemplate 默认 JSON 模板。
const defaultWebhookTemplate = `{"title":"{{.Title}}","content":"{{.Content}}","severity":"{{.Severity}}","ts":{{.Time}}}`

type webhookConfig struct {
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	BodyTemplate  string            `json:"body_template"`
}

type webhookNotifier struct {
	cfg  webhookConfig
	tmpl *template.Template
}

func newWebhook(c *model.NotificationChannel) (Notifier, error) {
	var cfg webhookConfig
	if err := c.ConfigJSON.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("Webhook 配置解析失败: %w", err)
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("Webhook URL 不能为空")
	}
	if cfg.Method == "" {
		cfg.Method = http.MethodPost
	}
	if cfg.BodyTemplate == "" {
		cfg.BodyTemplate = defaultWebhookTemplate
	}
	tmpl, err := template.New("body").Parse(cfg.BodyTemplate)
	if err != nil {
		return nil, fmt.Errorf("body_template 解析失败: %w", err)
	}
	return &webhookNotifier{cfg: cfg, tmpl: tmpl}, nil
}

func (n *webhookNotifier) Send(ctx context.Context, msg *Message) error {
	var buf bytes.Buffer
	if err := n.tmpl.Execute(&buf, msg); err != nil {
		return fmt.Errorf("模板渲染失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, n.cfg.Method, n.cfg.URL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range n.cfg.Headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("Webhook 返回 %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
