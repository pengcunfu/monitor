package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"monitor/internal/model"
)

type serverchanConfig struct {
	SendKey string `json:"sendkey"`
}

type serverchanNotifier struct {
	cfg serverchanConfig
}

func newServerChan(c *model.NotificationChannel) (Notifier, error) {
	var cfg serverchanConfig
	if err := c.ConfigJSON.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("Server酱配置解析失败: %w", err)
	}
	if cfg.SendKey == "" {
		return nil, fmt.Errorf("Server酱 sendkey 不能为空")
	}
	return &serverchanNotifier{cfg: cfg}, nil
}

func (n *serverchanNotifier) Send(ctx context.Context, msg *Message) error {
	endpoint := fmt.Sprintf("https://sctapi.ftqq.com/%s.send", n.cfg.SendKey)
	form := url.Values{}
	form.Set("title", msg.Title)
	form.Set("desp", fmt.Sprintf("%s\n\n%s", msg.Content, msg.Severity))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
	// code=0 表示成功
	if code := responseCode(body, "code"); code != 0 {
		return fmt.Errorf("Server酱返回错误: %s", string(body))
	}
	return nil
}

// responseCode 解析 JSON 响应中指定 key 的数字值。
func responseCode(body []byte, key string) int {
	m := map[string]interface{}{}
	if err := json.Unmarshal(body, &m); err != nil {
		return -1
	}
	switch v := m[key].(type) {
	case float64:
		return int(v)
	default:
		return -1
	}
}
