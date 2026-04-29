---
name: web-search
description: >
  Search the web and fetch webpage contents for real-time information retrieval.
  Use when the user needs current data, documentation, news, or information beyond the local codebase.
---

# Web Search

Search the internet and retrieve webpage contents for up-to-date information that cannot be found in the local codebase.

## When to Activate
Use this skill when the user asks to:
- Look up current information online (API docs, release notes, news, announcements)
- Fetch and analyze a specific URL's contents (documentation pages, blog posts, articles)
- Research a topic that requires fresh, time-sensitive, or external data
- Compare options, libraries, frameworks, or approaches using latest information
- Answer questions about technology versions, API changes, or deprecations
- Verify claims or find authoritative sources on a subject
- Find examples, tutorials, or best practices from the broader community

## Required Capabilities

This skill requires the following capabilities from the agent platform:

1. **Web / Internet Search** — Ability to search the internet, web indexes, or search engines for web pages, documents, or information matching a textual query. Should return URLs with brief relevance snippets, titles, or descriptions that help identify the most promising results. Supports formulating queries with keywords, phrases, date filters, and domain constraints.

2. **Web Page Content Retrieval** — Ability to retrieve the full textual content of a web page given its URL. Should extract readable article/body text while filtering out navigation elements, advertisements, sidebars, footers, and other non-content markup. Handles HTML-to-markdown conversion to produce clean, readable output suitable for analysis.

## Workflow

### 1. Search First
- Formulate search queries with specific, targeted keywords. Include version numbers, technology names, and context terms for technical accuracy.
- Execute web search to find relevant pages matching the query.
- Review search results to identify the most relevant and authoritative URLs. Prioritize official documentation over third-party blogs, recent sources over outdated ones, and primary sources over secondary summaries.
- If initial results are insufficient, refine the query: try alternative terminology, add or remove constraints, include date filters for recency, or narrow to specific domains.

### 2. Fetch Details
- For the most promising URLs identified from search, retrieve the full page content to extract detailed information.
- Focus extraction on the specific sections needed: code examples, API signatures, configuration samples, changelog entries, or explanation passages.
- When a page is very long, identify and focus on the relevant sections rather than processing the entire page uniformly.
- Cross-reference information across multiple sources if conflicting or incomplete information is found. Triangulate to determine the most accurate answer.

### 3. Synthesize and Cite
- Combine findings from multiple sources into a clear, well-structured answer that directly addresses the user's question.
- Cite sources appropriately — reference the origin URL and publication/recency date so the user can verify and follow up.
- Note any uncertainty, caveats, or temporal limitations (information may change, APIs may be deprecated, versions may have been superseded).
- Distinguish between factual information (from authoritative docs) and opinion/advice (from community discussions).

## Tips
- Start with broad search queries, then narrow down with targeted page fetching on the most promising results.
- Always include version numbers in search queries when researching technical topics — APIs and behaviors change between versions.
- For API documentation and technical references, prefer official vendor documentation over third-party tutorials, blog posts, or forum discussions.
- When a fetched page is lengthy, focus content retrieval on extracting the specific sections relevant to the question rather than processing the entire page.
- Always verify information currency — note publication dates, last-updated timestamps, and version applicability. Information about rapidly evolving technologies can become stale quickly.
- If search results seem low quality or irrelevant, try reformulating the query with synonyms or different conceptual framing before concluding the information isn't available.
