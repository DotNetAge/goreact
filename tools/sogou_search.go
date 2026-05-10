package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// --- Sogou (GouGou) Search Adapter ---

// SogouAdapter implements SearchAdapter using Sogou HTML search.
// 搜狗搜索引擎，提供中文内容优化。
type SogouAdapter struct {
	client *http.Client
}

// NewSogouAdapter creates a new Sogou search adapter.
func NewSogouAdapter() *SogouAdapter {
	return &SogouAdapter{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (a *SogouAdapter) Name() string { return "sogou" }

func (a *SogouAdapter) Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("page", "1")
	num := opts.MaxResults
	if num > 10 {
		num = 10
	}
	params.Set("num", strconv.Itoa(num))

	reqURL := "https://www.sogou.com/web?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	results := parseSogouHTML(body)
	return filterResults(results, opts), nil
}

// parseSogouHTML extracts search results from Sogou HTML response.
//
// 搜索结果HTML结构：
//
//	<div class="vrwrap">
//	    <h3 class="vr-title"><a href="URL" ...>TITLE</a></h3>
//	    <p class="str-text-info">SNIPPET</p>
//	    <span class="str-info">DISPLAY_URL</span>
//	</div>
func parseSogouHTML(html []byte) []SearchResult {
	content := string(html)
	var results []SearchResult

	parts := strings.Split(content, `class="vrwrap"`)
	if len(parts) <= 1 {
		parts = strings.Split(content, `class="rb"`)
	}
	if len(parts) <= 1 {
		return nil
	}

	for _, part := range parts[1:] {
		title, href := extractSogouTitle(part)
		snippet := extractSogouSnippet(part)

		if title == "" || href == "" {
			continue
		}

		finalURL := resolveSogouURL(href)
		if finalURL == "" {
			continue
		}

		results = append(results, SearchResult{
			Title:   htmlUnescape(stripTags(title)),
			URL:     finalURL,
			Snippet: htmlUnescape(stripTags(snippet)),
		})
	}

	return results
}

// extractSogouTitle extracts the title text and href from a result block.
func extractSogouTitle(part string) (title, href string) {
	h3Start := strings.Index(part, `<h3`)
	if h3Start < 0 {
		h3Start = strings.Index(part, `<h4`)
	}
	if h3Start < 0 {
		return "", ""
	}
	h3Part := part[h3Start:]
	h3End := strings.Index(h3Part, `</h3>`)
	if h3End < 0 {
		h3End = strings.Index(h3Part, `</h4>`)
	}
	if h3End < 0 {
		h3End = len(h3Part)
	}

	aPart := h3Part[:h3End]

	if idx := strings.Index(aPart, `href="`); idx >= 0 {
		hrefPart := aPart[idx+6:]
		if end := strings.Index(hrefPart, `"`); end >= 0 {
			href = htmlUnescape(hrefPart[:end])
		}
	}
	if idx := strings.Index(aPart, `href='`); idx >= 0 && href == "" {
		hrefPart := aPart[idx+6:]
		if end := strings.Index(hrefPart, `'`); end >= 0 {
			href = htmlUnescape(hrefPart[:end])
		}
	}

	if gt := strings.Index(aPart, ">"); gt >= 0 {
		textPart := aPart[gt+1:]
		if end := strings.Index(textPart, "</a>"); end >= 0 {
			title = strings.TrimSpace(textPart[:end])
		} else {
			title = strings.TrimSpace(textPart)
		}
	}

	return title, href
}

// extractSogouSnippet extracts the snippet from a result block.
func extractSogouSnippet(part string) string {
	for _, class := range []string{
		`class="str-text-info"`,
		`class="str_info"`,
		`class="fb"`,
	} {
		if idx := strings.Index(part, class); idx >= 0 {
			snippetPart := part[idx+len(class):]
			if gt := strings.Index(snippetPart, ">"); gt >= 0 {
				snippetPart = snippetPart[gt+1:]
				endTags := []string{"</p>", "</div>", "</span>"}
				for _, endTag := range endTags {
					if end := strings.Index(snippetPart, endTag); end >= 0 {
						return strings.TrimSpace(snippetPart[:end])
					}
				}
				return strings.TrimSpace(snippetPart)
			}
		}
	}
	return ""
}

// resolveSogouURL resolves the final URL from Sogou search result href.
func resolveSogouURL(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		if realURL := decodeSogouRedirectURL(href); realURL != "" {
			return realURL
		}
		return href
	}
	return ""
}

// decodeSogouRedirectURL extracts the real URL from a Sogou redirect link if present.
func decodeSogouRedirectURL(href string) string {
	if !strings.Contains(href, "sogou.com/link") && !strings.Contains(href, "sogou.com/web") {
		return ""
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	encodedURL := u.Query().Get("url")
	if encodedURL == "" {
		return ""
	}
	decoded, err := url.QueryUnescape(encodedURL)
	if err != nil {
		return ""
	}
	if strings.HasPrefix(decoded, "http://") || strings.HasPrefix(decoded, "https://") {
		return decoded
	}
	return ""
}
