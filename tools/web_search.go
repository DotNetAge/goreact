package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// --- WebSearch Tool (Claude-style adapter pattern) ---

// SearchResult represents a single web search result.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
}

// SearchAdapter is the interface for web search providers.
// Following Claude Code's adapter factory pattern, multiple providers
// can be registered and the system falls back through them.
type SearchAdapter interface {
	// Name returns the adapter's identifier.
	Name() string
	// Search performs a web search and returns results.
	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	MaxResults     int
	AllowedDomains []string
	BlockedDomains []string
}

// --- DuckDuckGo Adapter (zero-config fallback, like Claude's Bing adapter) ---

// DuckDuckGoAdapter implements SearchAdapter using DuckDuckGo HTML search.
// This is the zero-configuration fallback (always available, no API key needed),
// similar to Claude Code's BingSearchAdapter.
type DuckDuckGoAdapter struct {
	client *http.Client
}

// NewDuckDuckGoAdapter creates a new DuckDuckGo search adapter.
func NewDuckDuckGoAdapter() *DuckDuckGoAdapter {
	return &DuckDuckGoAdapter{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *DuckDuckGoAdapter) Name() string { return "duckduckgo" }

func (a *DuckDuckGoAdapter) Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("kl", "wt-wt") // no region bias
	params.Set("num", strconv.Itoa(opts.MaxResults))

	reqURL := "https://html.duckduckgo.com/html/?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	results := parseDuckDuckGoHTML(body)
	return filterResults(results, opts), nil
}

// parseDuckDuckGoHTML extracts search results from DuckDuckGo HTML response.
// DuckDuckGo HTML results use class="result" containers with anchor links.
func parseDuckDuckGoHTML(html []byte) []SearchResult {
	content := string(html)
	var results []SearchResult

	// DuckDuckGo HTML format: each result is in a <a class="result__a" href="...">
	// and the snippet is in <a class="result__snippet">
	type resultEntry struct {
		Title   string
		URL     string
		Snippet string
	}

	// Simple parsing strategy: find all result blocks
	parts := strings.Split(content, `<a rel="nofollow" class="result__a"`)
	for _, part := range parts[1:] {
		entry := resultEntry{}

		// Extract URL from href
		if idx := strings.Index(part, `href="`); idx >= 0 {
			hrefPart := part[idx+6:]
			if end := strings.Index(hrefPart, `"`); end >= 0 {
				entry.URL = htmlUnescape(hrefPart[:end])
				// DuckDuckGo uses redirect URLs, extract actual URL
				if uddg := strings.Index(entry.URL, "uddg="); uddg >= 0 {
					encoded := entry.URL[uddg+5:]
					if amp := strings.Index(encoded, "&"); amp >= 0 {
						encoded = encoded[:amp]
					}
					if decoded, err := url.QueryUnescape(encoded); err == nil {
						entry.URL = decoded
					}
				}
			}
		}

		// Extract title (text between > and </a>)
		if gt := strings.Index(part, ">"); gt >= 0 {
			if end := strings.Index(part[gt:], "</a>"); end >= 0 {
				entry.Title = htmlUnescape(strings.TrimSpace(part[gt+1 : gt+end]))
			}
		}

		// Extract snippet
		if idx := strings.Index(part, `class="result__snippet"`); idx >= 0 {
			snippetPart := part[idx:]
			if gt := strings.Index(snippetPart, ">"); gt >= 0 {
				snippetPart = snippetPart[gt+1:]
				if end := strings.Index(snippetPart, "</a>"); end >= 0 {
					entry.Snippet = htmlUnescape(strings.TrimSpace(snippetPart[:end]))
				}
			}
		}

		if entry.Title != "" && entry.URL != "" {
			results = append(results, SearchResult{
				Title:   entry.Title,
				URL:     entry.URL,
				Snippet: entry.Snippet,
			})
		}
	}

	return results
}

func htmlUnescape(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return s
}

func filterResults(results []SearchResult, opts SearchOptions) []SearchResult {
	if opts.MaxResults > 0 && len(results) > opts.MaxResults {
		results = results[:opts.MaxResults]
	}

	filtered := make([]SearchResult, 0, len(results))
	for _, r := range results {
		u, err := url.Parse(r.URL)
		if err != nil {
			continue
		}
		// Check blocked domains
		blocked := false
		for _, d := range opts.BlockedDomains {
			if strings.HasSuffix(u.Hostname(), d) {
				blocked = true
				break
			}
		}
		if blocked {
			continue
		}
		// Check allowed domains (if specified)
		if len(opts.AllowedDomains) > 0 {
			allowed := false
			for _, d := range opts.AllowedDomains {
				if strings.HasSuffix(u.Hostname(), d) {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// --- WebSearchTool ---

// WebSearchTool performs web searches, returning {title, url} pairs.
// Following Claude Code's architecture: search discovers URLs, WebFetch reads them.
//
// Claude Code pattern:
//   - WebSearch: lightweight discovery → returns {title, url} only (small token cost)
//   - WebFetch: deep reading → local HTTP fetch → HTML→Markdown → LLM summarization
type WebSearchTool struct {
	adapters []SearchAdapter
	cache    sync.Map // map[string]cachedSearch
	cacheTTL time.Duration
}

type cachedSearch struct {
	results   []SearchResult
	timestamp time.Time
}

// NewWebSearchTool creates a WebSearchTool with DuckDuckGo as default adapter.
func NewWebSearchTool() core.FuncTool {
	t := &WebSearchTool{
		cacheTTL: 15 * time.Minute,
	}
	// Register default adapter (always available, no API key needed)
	t.adapters = append(t.adapters, NewDuckDuckGoAdapter())
	return t
}

// AddAdapter adds a search adapter to the fallback chain.
// Adapters are tried in order; first successful result wins.
func (t *WebSearchTool) AddAdapter(adapter SearchAdapter) {
	t.adapters = append([]SearchAdapter{adapter}, t.adapters...)
}

func (t *WebSearchTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name: "web_search",
		Description: `Search the web for real-time information. Returns a list of {title, url} results.
Use this tool when you need up-to-date information beyond your training data.

Key behaviors:
- Returns only {title, url} pairs (small token cost).
- Use 'web_fetch' to read the full content of any interesting URL.
- Supports domain filtering via allowed_domains and blocked_domains.
- Results are cached for 15 minutes to avoid redundant searches.

Parameters:
- query: the search query string (required)
- max_results: maximum number of results to return (default: 10)
- allowed_domains: restrict results to these domains (optional)
- blocked_domains: exclude results from these domains (optional)`,
		IsReadOnly: true,
		Parameters: []core.Parameter{
			{
				Name:        "query",
				Type:        "string",
				Description: "The search query string. Be specific and include relevant keywords.",
				Required:    true,
			},
			{
				Name:        "max_results",
				Type:        "integer",
				Description: "Maximum number of results to return (default: 10, max: 20).",
				Required:    false,
			},
			{
				Name:        "allowed_domains",
				Type:        "array",
				Description: "Restrict results to these domains (e.g., [\"github.com\", \"docs.python.org\"]).",
				Required:    false,
			},
			{
				Name:        "blocked_domains",
				Type:        "array",
				Description: "Exclude results from these domains.",
				Required:    false,
			},
		},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	query, err := ValidateRequiredString(params, "query")
	if err != nil {
		return nil, err
	}
	if len(query) < 2 {
		return nil, fmt.Errorf("query must be at least 2 characters")
	}

	maxResults := 10
	if raw, ok := params["max_results"]; ok {
		if v, ok := ToFloat64(raw); ok && v > 0 {
			maxResults = int(v)
			if maxResults > 20 {
				maxResults = 20
			}
		}
	}

	var allowedDomains, blockedDomains []string
	if raw, ok := params["allowed_domains"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				allowedDomains = append(allowedDomains, s)
			}
		}
	}
	if raw, ok := params["blocked_domains"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				blockedDomains = append(blockedDomains, s)
			}
		}
	}

	opts := SearchOptions{
		MaxResults:     maxResults,
		AllowedDomains: allowedDomains,
		BlockedDomains: blockedDomains,
	}

	// Check cache
	cacheKey := query + "|" + strings.Join(allowedDomains, ",") + "|" + strings.Join(blockedDomains, ",")
	if cached, ok := t.cache.Load(cacheKey); ok {
		entry := cached.(cachedSearch)
		if time.Since(entry.timestamp) < t.cacheTTL {
			return formatSearchResults(query, entry.results), nil
		}
		t.cache.Delete(cacheKey)
	}

	// Try adapters in order (fallback chain)
	var results []SearchResult
	var lastErr error
	for _, adapter := range t.adapters {
		searchResults, err := adapter.Search(ctx, query, opts)
		if err != nil {
			lastErr = err
			continue
		}
		if len(searchResults) > 0 {
			results = searchResults
			break
		}
	}

	if len(results) == 0 {
		if lastErr != nil {
			return nil, fmt.Errorf("all search adapters failed: %w", lastErr)
		}
		return fmt.Sprintf("No results found for query: %q", query), nil
	}

	// Cache results
	t.cache.Store(cacheKey, cachedSearch{
		results:   results,
		timestamp: time.Now(),
	})

	return formatSearchResults(query, results), nil
}

func formatSearchResults(query string, results []SearchResult) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Web search results for query: %q\n\n", query)

	for i, r := range results {
		fmt.Fprintf(&sb, "%d. [%s](%s)\n", i+1, r.Title, r.URL)
		if r.Snippet != "" {
			snippet := r.Snippet
			if len([]rune(snippet)) > 200 {
				snippet = string([]rune(snippet)[:200]) + "..."
			}
			fmt.Fprintf(&sb, "   %s\n", snippet)
		}
	}

	fmt.Fprintln(&sb, "\nUse 'web_fetch' to read the full content of any URL above.")
	return sb.String()
}

// --- WebFetch Tool (Claude-style: local fetch → content extraction) ---

// WebFetchToolClaude implements Claude-style WebFetch:
// local HTTP fetch → content processing → focused answer.
//
// Claude Code pattern:
//   1. URL validation + SSRF protection
//   2. Local HTTP fetch (not server-side)
//   3. HTML → text extraction (strips scripts, styles, etc.)
//   4. Returns processed content with metadata
//
// Note: Claude Code uses a secondary LLM (Haiku) for summarization.
// Since this is a Go implementation without a guaranteed small model,
// we do intelligent content extraction instead.
type WebFetchToolClaude struct {
	client  *http.Client
	cache   sync.Map // map[string]cachedFetch
	cacheTTL time.Duration
}

type cachedFetch struct {
	content   string
	timestamp time.Time
}

// NewWebFetchToolClaude creates a Claude-style WebFetch tool.
func NewWebFetchToolClaude() core.FuncTool {
	return &WebFetchToolClaude{
		client: &http.Client{Timeout: 15 * time.Second},
		cacheTTL: 15 * time.Minute,
	}
}

func (t *WebFetchToolClaude) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name: "web_fetch",
		Description: `Fetch and extract content from a web page. Unlike web_search which discovers URLs, web_fetch reads the actual content of a known URL.

Claude-style architecture:
1. Validates URL and checks for SSRF risks (blocks private IPs).
2. Fetches the page content locally via HTTP.
3. Strips HTML tags (scripts, styles, nav, etc.) for clean text.
4. Returns the extracted content for the LLM to process.

Use this after web_search to read specific URLs that look relevant.

Parameters:
- url: the URL to fetch (required)
- prompt: what information to extract from the page (optional, helps focus the output)`,
		IsReadOnly: true,
		Parameters: []core.Parameter{
			{
				Name:        "url",
				Type:        "string",
				Description: "The URL to fetch content from.",
				Required:    true,
			},
			{
				Name:        "prompt",
				Type:        "string",
				Description: "What information to extract or what question to answer about the page content. Helps focus the output on relevant details.",
				Required:    false,
			},
		},
	}
}

func (t *WebFetchToolClaude) Execute(ctx context.Context, params map[string]any) (any, error) {
	rawURL, err := ValidateRequiredString(params, "url")
	if err != nil {
		return nil, err
	}
	prompt, _ := params["prompt"].(string)

	// Normalize URL
	rawURL = strings.TrimSpace(rawURL)
	if strings.HasPrefix(rawURL, "http://") {
		rawURL = "https://" + rawURL[7:]
	}
	if !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	// SSRF protection
	if err := validateURL(rawURL); err != nil {
		return nil, err
	}

	// Check cache
	if cached, ok := t.cache.Load(rawURL); ok {
		entry := cached.(cachedFetch)
		if time.Since(entry.timestamp) < t.cacheTTL {
			if prompt != "" {
				return fmt.Sprintf("--- Web Fetch: %s ---\nPrompt: %s\n\n%s", rawURL, prompt, entry.content), nil
			}
			return fmt.Sprintf("--- Web Fetch: %s ---\n\n%s", rawURL, entry.content), nil
		}
		t.cache.Delete(rawURL)
	}

	// Fetch
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,text/plain;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Process content
	content := htmlToText(string(body))
	content = TruncateString(content, 50000) // 50K chars max

	// Cache
	t.cache.Store(rawURL, cachedFetch{
		content:   content,
		timestamp: time.Now(),
	})

	// Format output
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- Web Fetch: %s ---\n", rawURL)
	fmt.Fprintf(&sb, "Status: %d | Size: %d chars\n", resp.StatusCode, len(content))
	if prompt != "" {
		fmt.Fprintf(&sb, "Prompt: %s\n", prompt)
	}
	fmt.Fprintf(&sb, "\n%s\n", content)

	return sb.String(), nil
}

// htmlToText converts HTML to plain text, stripping tags and normalizing whitespace.
// This is a simplified version - a production implementation would use a proper
// HTML-to-Markdown converter (like Turndown in Claude Code).
func htmlToText(html string) string {
	// Remove script and style blocks
	html = stripTag(html, "script")
	html = stripTag(html, "style")
	html = stripTag(html, "nav")
	html = stripTag(html, "header")
	html = stripTag(html, "footer")
	html = stripTag(html, "noscript")

	// Convert common block elements to newlines
	html = blockToNewline(html, "p")
	html = blockToNewline(html, "div")
	html = blockToNewline(html, "br")
	html = blockToNewline(html, "h1")
	html = blockToNewline(html, "h2")
	html = blockToNewline(html, "h3")
	html = blockToNewline(html, "h4")
	html = blockToNewline(html, "h5")
	html = blockToNewline(html, "h6")
	html = blockToNewline(html, "li")
	html = blockToNewline(html, "tr")
	html = blockToNewline(html, "pre")
	html = blockToNewline(html, "blockquote")

	// Remove all remaining HTML tags
	result := stripAllTags(html)

	// Decode HTML entities
	result = htmlUnescape(result)

	// Normalize whitespace
	lines := strings.Split(result, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}

func stripTag(html, tag string) string {
	// Remove opening and closing tags and everything between them
	for {
		start := strings.Index(strings.ToLower(html), "<"+tag)
		if start == -1 {
			break
		}
		end := strings.Index(html[start:], ">")
		if end == -1 {
			break
		}
		closeTag := "</" + tag
		closeIdx := strings.Index(strings.ToLower(html[start:]), closeTag)
		if closeIdx == -1 {
			html = html[:start] + html[start+end+1:]
			continue
		}
		closeEnd := strings.Index(html[start+closeIdx:], ">")
		if closeEnd == -1 {
			html = html[:start] + html[start+end+1:]
			continue
		}
		html = html[:start] + html[start+closeIdx+closeEnd+1:]
	}
	return html
}

func blockToNewline(html, tag string) string {
	html = strings.ReplaceAll(html, "<"+tag+">", "\n")
	html = strings.ReplaceAll(html, "<"+tag+" ", "\n")
	html = strings.ReplaceAll(html, "</"+tag+">", "\n")
	return html
}

func stripAllTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// MarshalJSON for SearchResult
func (r SearchResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet,omitempty"`
	}{
		Title:   r.Title,
		URL:     r.URL,
		Snippet: r.Snippet,
	})
}
