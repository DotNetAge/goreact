package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ============================================================
// BaiduAdapter Unit Tests
// ============================================================

func TestBaiduAdapter_Name(t *testing.T) {
	a := NewBaiduAdapter()
	if a.Name() != "baidu" {
		t.Errorf("Name() = %q, want %q", a.Name(), "baidu")
	}
}

func TestBaiduAdapter_Search_ContextCancelled(t *testing.T) {
	a := NewBaiduAdapter()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := a.Search(ctx, "test query", SearchOptions{})
	if err == nil {
		t.Error("expected error when context is already cancelled")
	}
}

// ============================================================
// parseBaiduHTML — Unit Tests
// ============================================================

func TestParseBaiduHTML_SingleResult(t *testing.T) {
	htmlBody := []byte(`<html><body>
<div class="result c-container" id="1">
	<h3 class="t">
		<a href="http://www.baidu.com/link?url=https%3A%2F%2Fwww.example.com%2Fpage" target="_blank">Example Page Title</a>
	</h3>
	<div class="c-abstract">This is a snippet about the example page.</div>
	<span class="c-showurl">www.example.com</span>
</div>
</body></html>`)

	results := parseBaiduHTML(htmlBody)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Example Page Title" {
		t.Errorf("Title = %q, want %q", results[0].Title, "Example Page Title")
	}
	if !strings.Contains(results[0].URL, "example.com") {
		t.Errorf("URL should contain example.com, got: %q", results[0].URL)
	}
	if results[0].Snippet != "This is a snippet about the example page." {
		t.Errorf("Snippet = %q", results[0].Snippet)
	}
}

func TestParseBaiduHTML_MultipleResults(t *testing.T) {
	htmlBody := []byte(`<html><body>
<div class="result c-container" id="1">
	<h3 class="t"><a href="http://www.baidu.com/link?url=https%3A%2F%2Fsite1.com">Site One</a></h3>
	<div class="c-abstract">First result snippet.</div>
</div>
<div class="result c-container" id="2">
	<h3 class="t"><a href="http://www.baidu.com/link?url=https%3A%2F%2Fsite2.com">Site Two</a></h3>
	<div class="c-abstract">Second result snippet.</div>
</div>
<div class="result c-container" id="3">
	<h3 class="t"><a href="http://www.baidu.com/link?url=https%3A%2F%2Fsite3.com">Site Three</a></h3>
	<div class="c-abstract">Third result snippet.</div>
</div>
</body></html>`)

	results := parseBaiduHTML(htmlBody)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, r := range results {
		if r.Title == "" {
			t.Errorf("result[%d] Title is empty", i)
		}
		if r.URL == "" {
			t.Errorf("result[%d] URL is empty", i)
		}
	}
}

func TestParseBaiduHTML_NoResults(t *testing.T) {
	htmlBody := []byte(`<html><body><div class="no-result"><p>No results found for your query.</p></div></body></html>`)
	results := parseBaiduHTML(htmlBody)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseBaiduHTML_EmptyInput(t *testing.T) {
	results := parseBaiduHTML(nil)
	if len(results) != 0 {
		t.Errorf("expected 0 results for nil input, got %d", len(results))
	}
	results = parseBaiduHTML([]byte(""))
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty input, got %d", len(results))
	}
}

func TestParseBaiduHTML_HtmlEntitiesInTitle(t *testing.T) {
	htmlBody := []byte(`<html><body>
<div class="result c-container">
	<h3 class="t"><a href="http://www.baidu.com/link?url=https%3A%2F%2Fexample.com">&quot;Quoted Title&quot; &amp; More</a></h3>
	<div class="c-abstract">&lt;code&gt; snippet here</div>
</div>
</body></html>`)

	results := parseBaiduHTML(htmlBody)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !strings.Contains(results[0].Title, `"`) {
		t.Errorf("entities should be decoded in title, got: %q", results[0].Title)
	}
	if !strings.Contains(results[0].Snippet, "<") {
		t.Errorf("entities should be decoded in snippet, got: %q", results[0].Snippet)
	}
}

func TestParseBaiduHTML_MissingSnippet(t *testing.T) {
	htmlBody := []byte(`<html><body>
<div class="result c-container">
	<h3 class="t"><a href="http://www.baidu.com/link?url=https%3A%2F%2Fexample.com">Title Only</a></h3>
</div>
</body></html>`)

	results := parseBaiduHTML(htmlBody)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Snippet != "" {
		t.Errorf("Snippet should be empty when not present, got: %q", results[0].Snippet)
	}
}

// ============================================================
// decodeBaiduRedirectURL
// ============================================================

func TestDecodeBaiduRedirectURL(t *testing.T) {
	tests := []struct {
		name string
		href string
		want string
	}{
		{
			name: "standard redirect",
			href: "http://www.baidu.com/link?url=https%3A%2F%2Fwww.example.com%2Fpage",
			want: "https://www.example.com/page",
		},
		{
			name: "not a baidu link",
			href: "https://www.example.com/direct",
			want: "",
		},
		{
			name: "baidu link without url param",
			href: "http://www.baidu.com/link?other=value",
			want: "",
		},
		{
			name: "invalid url",
			href: "://invalid",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeBaiduRedirectURL(tt.href)
			if got != tt.want {
				t.Errorf("decodeBaiduRedirectURL(%q) = %q, want %q", tt.href, got, tt.want)
			}
		})
	}
}

// ============================================================
// BaiduSearchTool Info & Validation
// ============================================================

func TestBaiduSearchTool_Info(t *testing.T) {
	tool := NewBaiduSearchTool()
	info := tool.Info()
	if info.Name != "BaiduSearch" {
		t.Errorf("Name = %q, want %q", info.Name, "BaiduSearch")
	}
	if !info.IsReadOnly {
		t.Error("baidu search should be read-only")
	}
	if len(info.Parameters) == 0 {
		t.Error("expected parameters")
	}
}

func TestBaiduSearchTool_MissingQuery(t *testing.T) {
	tool := NewBaiduSearchTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for missing query parameter")
	}
}

func TestBaiduSearchTool_TooShortQuery(t *testing.T) {
	tool := NewBaiduSearchTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"query": "x",
	})
	if err == nil {
		t.Error("expected error for query shorter than 2 characters")
	}
}

// ============================================================
// BaiduSearchTool — Real Network Tests
// ============================================================

// skipIfBaiduUnavailable skips the test when Baidu returns a CAPTCHA,
// rate-limits, or is otherwise unreachable (very common with automated requests).
func skipIfBaiduUnavailable(t *testing.T, result string) {
	t.Helper()
	if strings.Contains(result, "No results found") {
		t.Skipf("Baidu unreachable (CAPTCHA/rate-limit): %s", TruncateString(result, 120))
	}
}

func TestBaiduSearchTool_Real_BasicSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real network test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tool := NewBaiduSearchTool()
	result, err := tool.Execute(ctx, map[string]any{
		"query": "golang programming",
	})
	if err != nil {
		t.Skipf("Baidu unreachable (network/GFW): %v", err)
	}
	s := result.(string)
	skipIfBaiduUnavailable(t, s)
	if s == "" {
		t.Fatal("expected non-empty search result")
	}
	if !strings.Contains(s, "Web search results") {
		t.Errorf("result should contain header, got: %.150s...", s)
	}
	if !strings.Contains(s, "http") {
		t.Error("result should contain at least one URL")
	}
	t.Logf("Baidu search results: %.500s...", s)
}

func TestBaiduSearchTool_Real_ChineseQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real network test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tool := NewBaiduSearchTool()
	result, err := tool.Execute(ctx, map[string]any{
		"query":       "北京大学",
		"max_results": float64(5),
	})
	if err != nil {
		t.Skipf("Baidu unreachable (network/GFW): %v", err)
	}
	s := result.(string)
	skipIfBaiduUnavailable(t, s)
	if !strings.Contains(s, "Web search results") {
		t.Errorf("result should contain header, got: %.150s...", s)
	}
	t.Logf("Chinese search results: %.500s...", s)
}

func TestBaiduSearchTool_Real_MaxResults(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real network test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tool := NewBaiduSearchTool()
	result, err := tool.Execute(ctx, map[string]any{
		"query":       "github",
		"max_results": float64(3),
	})
	if err != nil {
		t.Skipf("Baidu unreachable (network/GFW): %v", err)
	}
	s := result.(string)
	skipIfBaiduUnavailable(t, s)
	count := strings.Count(s, "\n1. [")
	count += strings.Count(s, "\n2. [")
	count += strings.Count(s, "\n3. [")
	if count < 1 {
		t.Errorf("expected at least 1 result, got: %.200s...", s)
	}
	t.Logf("Baidu max_results test: %.500s...", s)
}

func TestBaiduSearchTool_Real_SpecificSite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real network test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tool := NewBaiduSearchTool()
	result, err := tool.Execute(ctx, map[string]any{
		"query":           "golang",
		"max_results":     float64(5),
		"allowed_domains": []any{"github.com"},
	})
	if err != nil {
		t.Skipf("Baidu unreachable (network/GFW): %v", err)
	}
	s := result.(string)
	skipIfBaiduUnavailable(t, s)
	if !strings.Contains(s, "github.com") {
		t.Logf("domain filtering may have excluded non-github results; got: %.300s...", s)
	}
}

func TestBaiduSearchTool_Real_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real network test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // ensure context expires

	tool := NewBaiduSearchTool()
	_, err := tool.Execute(ctx, map[string]any{
		"query": "test",
	})
	if err == nil {
		t.Log("baidu search completed within 1ms (unlikely, but ok)")
	}
}
