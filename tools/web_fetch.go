package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// WebFetchTool implements a tool for fetching web content.
type WebFetchTool struct{}

// NewWebFetchTool 创建网页抓取工具
func NewWebFetchTool() core.FuncTool {
	return &WebFetchTool{}
}

func (t *WebFetchTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "web_fetch",
		Description: "Fetch the content of a URL. Returns the raw content (HTML/Text).",
		Parameters: []core.Parameter{
			{
				Name:        "url",
				Type:        "string",
				Description: "The URL to fetch.",
				Required:    true,
			},
		},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing url parameter")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	content := string(body)
	// Truncate if too large
	if len(content) > 50000 {
		content = content[:50000] + "\n... [content truncated] ..."
	}

	return content, nil
}
