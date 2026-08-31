package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// httpClient 共享 HTTP 客户端（10s 超时）。
var httpClient = &http.Client{Timeout: 10 * time.Second}

// postJSON POST JSON 载荷到 url，校验响应中 codeKey 字段为 0。
// 失败重试 1 次；响应体截断返回给上层。
func postJSON(ctx context.Context, url string, payload interface{}, codeKey string) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		_ = resp.Body.Close()
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		code, _ := result[codeKey].(float64)
		if code != 0 {
			lastErr = fmt.Errorf("远端返回错误: %s", string(body))
			continue
		}
		return nil
	}
	return lastErr
}
