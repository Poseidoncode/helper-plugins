# SKILL: web-crawler

## Description
High-performance web scraping and crawling engine powered by Colly, Goquery, and Chromedp (Headless Chrome). Provides dual-path extraction: ultra-fast static HTML scraping/Markdown conversion, and full JavaScript/SPA rendering with DOM interaction and screenshots.

## Tools Provided

### 1. `scrape_static`
- **When to use**: Fast scraping of documentation, technical blogs, news articles, GitHub pages, and standard HTML.
- **Key Parameters**:
  - `url`: Target web page URL.
  - `selector` (optional): CSS selector to restrict extraction to a specific container (e.g. `article`, `main`, `#readme`).
  - `extract_links` (optional): Boolean flag to extract unique hyperlinks.
  - `convert_markdown` (optional, default `true`): Converts extracted HTML into clean Markdown.

### 2. `scrape_dynamic`
- **When to use**: Web pages that require client-side JavaScript execution (SPAs, React/Vue/Angular apps), pages with delayed data hydration, or when a full-page screenshot is needed.
- **Key Parameters**:
  - `url`: Target web page URL.
  - `wait_selector` (optional): CSS selector to wait for before extracting (ensures dynamic data has rendered).
  - `wait_seconds` (optional): Additional pause for JS hydration.
  - `extract_selector` (optional): CSS selector to scope extraction.
  - `screenshot_path` (optional): File path to save a full-page PNG screenshot.
  - `convert_markdown` (optional, default `true`): Converts rendered DOM into Markdown.

### 3. `crawl_site`
- **When to use**: Bounded multi-page discovery across a documentation hierarchy or website domain.
- **Key Parameters**:
  - `start_url`: Entry URL to start crawling.
  - `allowed_domains` (optional): Hostnames to restrict crawling within.
  - `max_depth` (optional, default `1`, max `3`): Link crawl depth.
  - `max_pages` (optional, default `10`, max `50`): Maximum total pages visited.
  - `rate_limit_ms` (optional, default `500`): Politeness delay between requests.
