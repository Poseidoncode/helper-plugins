# Web Crawler & Dynamic Scraper Plugin (`web-crawler`)

High-performance web crawling and scraping plugin for the [Helper](https://github.com/Poseidoncode/Helper) ecosystem, combining **Colly**, **Goquery**, and **Chromedp**.

## 🚀 Features

- ⚡ **Ultra-fast Static Scraping (`scrape_static`)**: Colly + Goquery with automatic HTML-to-Markdown conversion.
- 🌐 **Headless Browser Rendering (`scrape_dynamic`)**: Direct Chrome DevTools Protocol integration via Chromedp for SPAs, JS hydration, and full-page screenshots.
- 🕸️ **Bounded Multi-page Crawling (`crawl_site`)**: Controlled depth, domain bounding, and polite rate limiting.
- 🔌 **Standard MCP Protocol**: Stdio JSON-RPC 2.0 server ready for Helper, Claude, and AI agent integration.

## 📦 Installation & Usage

### 1. Build from source
```bash
cd plugins/web-crawler
go build -o web-crawler ./cmd/web-crawler
```

### 2. Standalone CLI usage
```bash
# Fast static fetch
./web-crawler fetch https://example.com

# Dynamic rendering
./web-crawler render https://example.com

# Bounded crawl
./web-crawler crawl https://example.com

# MCP stdio server
./web-crawler --stdio
```

## 🛠️ MCP Tools Overview

| Tool Name | Engine | Best Used For |
|---|---|---|
| `scrape_static` | Colly + Goquery | Documentation, blog posts, news, static HTML |
| `scrape_dynamic` | Chromedp (Headless Chrome) | React/Vue SPA apps, delayed JS loads, screenshots |
| `crawl_site` | Colly (Bounded) | Multi-page link discovery within domain |

## 📄 License
Apache-2.0
