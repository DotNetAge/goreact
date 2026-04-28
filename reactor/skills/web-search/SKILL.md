---
name: web-search
description: >
  Search the web and fetch webpage contents for real-time information retrieval.
  Use when the user needs current data, documentation, news, or information beyond the local codebase.
allowed-tools: web_search web_fetch
---

# Web Search

Search the internet and retrieve webpage contents for up-to-date information.

## When to Activate
Use this skill when the user asks to:
- Look up current information online (API docs, news, releases)
- Fetch and analyze a specific URL's contents
- Research a topic that requires fresh or external data
- Compare options or find latest best practices
- Answer questions about technologies, versions, or APIs

## Workflow

### 1. Search First
- Use `web_search` to find relevant pages. Formulate queries with specific keywords.
- Review search results to identify the most relevant URLs.
- If results are insufficient, refine the query with different terms or add date/context keywords.

### 2. Fetch Details
- Use `web_fetch` on promising URLs to extract full content.
- Extract the specific information needed (code examples, API signatures, configuration).
- Cross-reference multiple sources if conflicting information is found.

### 3. Synthesize
- Combine findings into a clear answer.
- Cite sources where appropriate.
- Note any uncertainty or dates (information may change).

## Tool Reference

| Tool | Best For | Example |
|------|----------|---------|
| web_search | Finding URLs and discovering information | `"GoReact agent framework 2025"`, `"golang context cancellation best practices"` |
| web_fetch | Reading full page content from a known URL | `https://pkg.dev/github.com/...` |

## Tips
- Start broad with `web_search`, then narrow down with targeted `web_fetch`.
- Include version numbers in search queries for technical accuracy.
- For API documentation, prefer official docs over third-party blogs.
- If a page is too long, focus `web_fetch` on extracting specific sections.
- Always verify information currency — note publication/recency dates.
