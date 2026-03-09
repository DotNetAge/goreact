package builtin

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ray/goreact/pkg/tool"
)

// HTTP HTTP 请求工具
type HTTP struct {
	client *http.Client
}

// NewHTTP 创建 HTTP 工具
func NewHTTP() tool.Tool {
	return &HTTP{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
			},
		},
	}
}

// Name 返回工具名称
func (h *HTTP) Name() string {
	return "http"
}

// Description 返回工具描述
func (h *HTTP) Description() string {
	return "Make HTTP requests. Params: {method: 'GET'|'POST'|'PUT'|'DELETE', url: 'http://...', body: 'request body'}"
}

// Execute 执行 HTTP 请求
func (h *HTTP) Execute(params map[string]interface{}) (interface{}, error) {
	method, ok := params["method"].(string)
	if !ok {
		method = "GET"
	}

	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'url' parameter")
	}

	// 构建请求
	var body io.Reader
	if bodyStr, ok := params["body"].(string); ok && bodyStr != "" {
		body = bytes.NewBufferString(bodyStr)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 添加请求头
	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// 发送请求
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 返回结果
	result := map[string]interface{}{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"body":        string(respBody),
		"headers":     resp.Header,
	}

	return result, nil
}
