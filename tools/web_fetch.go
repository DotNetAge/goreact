package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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

// isPrivateIP checks whether an IP address belongs to a private/reserved range.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{parseCIDR("127.0.0.0/8")},       // loopback
		{parseCIDR("10.0.0.0/8")},        // RFC 1918
		{parseCIDR("172.16.0.0/12")},     // RFC 1918
		{parseCIDR("192.168.0.0/16")},    // RFC 1918
		{parseCIDR("169.254.0.0/16")},    // link-local
		{parseCIDR("::1/128")},           // IPv6 loopback
		{parseCIDR("fc00::/7")},          // IPv6 ULA
		{parseCIDR("fe80::/10")},         // IPv6 link-local
		{parseCIDR("0.0.0.0/8")},         // current network
	}
	for _, r := range privateRanges {
		if r.network != nil && r.network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseCIDR(s string) *net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		return nil
	}
	return network
}

// validateURL checks that a URL is not pointing to a private/internal address (SSRF protection).
func validateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s (only http/https allowed)", parsed.Scheme)
	}

	host := parsed.Hostname()
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("failed to resolve host %q: %w", host, err)
	}
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("access denied: URL resolves to private/internal address %s", ip)
		}
	}
	return nil
}

func (t *WebFetchTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing url parameter")
	}

	// SSRF protection: reject private/internal addresses
	if err := validateURL(url); err != nil {
		return nil, err
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
	if len([]rune(content)) > 50000 {
		runes := []rune(content)
		content = string(runes[:50000]) + "\n... [content truncated] ..."
	}

	return content, nil
}
