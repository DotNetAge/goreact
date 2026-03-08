package builtin

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// Curl curl工具
type Curl struct{}

// NewCurl 创建curl工具
func NewCurl() *Curl {
	return &Curl{}
}

// Name 返回工具名称
func (c *Curl) Name() string {
	return "curl"
}

// Description 返回工具描述
func (c *Curl) Description() string {
	return "HTTP请求工具，支持发送GET、POST等HTTP请求"
}

// Execute 执行curl操作
func (c *Curl) Execute(params map[string]interface{}) (interface{}, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'url' parameter")
	}

	method, ok := params["method"].(string)
	if !ok {
		method = "GET"
	}

	headers, _ := params["headers"].(map[string]interface{})
	body, _ := params["body"].(string)

	// 创建请求
	req, err := http.NewRequest(strings.ToUpper(method), url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	if headers != nil {
		for key, value := range headers {
			if valStr, ok := value.(string); ok {
				req.Header.Set(key, valStr)
			}
		}
	}

	// 设置默认Content-Type
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 构建结果
	result := map[string]interface{}{
		"status":   resp.Status,
		"statusCode": resp.StatusCode,
		"body":     string(respBody),
		"headers":  resp.Header,
	}

	return result, nil
}
