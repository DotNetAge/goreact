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

// --- 360 (Haosou) Search Adapter ---

// HaosouAdapter implements SearchAdapter using 360 Haosou HTML search.
// 360搜索是中国主要的搜索引擎之一，提供中文内容优化。
type HaosouAdapter struct {
	client *http.Client
}

// NewHaosouAdapter creates a new 360 Haosou search adapter.
func NewHaosouAdapter() *HaosouAdapter {
	return &HaosouAdapter{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (a *HaosouAdapter) Name() string { return "haosou" }

func (a *HaosouAdapter) Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("pn", "0")
	params.Set("ps", strconv.Itoa(opts.MaxResults))

	reqURL := "https://www.so.com/s?" + params.Encode()
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

	results := parseHaosouHTML(body)
	return filterResults(results, opts), nil
}

// parseHaosouHTML extracts search results from 360 Haosou HTML response.
//
// 360搜索结果HTML结构：
//
//	<div class="res-list" data-res="...">
//	    <h3 class="res-title"><a href="URL" ...>TITLE</a></h3>
//	    <p class="res-desc">SNIPPET</p>
//	    <span class="res-site">DISPLAY_URL</span>
//	</div>
func parseHaosouHTML(html []byte) []SearchResult {
	content := string(html)
	var results []SearchResult

	parts := strings.Split(content, `class="res-list"`)
	if len(parts) <= 1 {
		return nil
	}

	for _, part := range parts[1:] {
		title, href := extractHaosouTitle(part)
		snippet := extractHaosouSnippet(part)

		if title == "" || href == "" {
			continue
		}

		finalURL := resolveHaosouURL(href)
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

// extractHaosouTitle extracts the title text and href from a result block.
func extractHaosouTitle(part string) (title, href string) {
	h3Start := strings.Index(part, `<h3`)
	if h3Start < 0 {
		return "", ""
	}
	h3Part := part[h3Start:]
	h3End := strings.Index(h3Part, `</h3>`)
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

// extractHaosouSnippet extracts the snippet from a result block.
func extractHaosouSnippet(part string) string {
	for _, class := range []string{
		`class="res-desc"`,
		`class="res-desc-info"`,
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

// resolveHaosouURL resolves the final URL from 360 search result href.
// 360 may use redirect URLs that need to be resolved.
func resolveHaosouURL(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		if realURL := decodeHaosouRedirectURL(href); realURL != "" {
			return realURL
		}
		return href
	}
	return ""
}

// decodeHaosouRedirectURL extracts the real URL from a 360 redirect link if present.
func decodeHaosouRedirectURL(href string) string {
	if !strings.Contains(href, "so.com/link") && !strings.Contains(href, "haoso.com") {
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
