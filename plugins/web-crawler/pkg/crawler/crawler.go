package crawler

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// CrawlOptions defines options for bounded multi-page crawling.
type CrawlOptions struct {
	StartURL        string   `json:"start_url"`
	AllowedDomains  []string `json:"allowed_domains,omitempty"`
	MaxDepth        int      `json:"max_depth,omitempty"`
	MaxPages        int      `json:"max_pages,omitempty"`
	RateLimitMs     int      `json:"rate_limit_ms,omitempty"`
	ExtractSelector string   `json:"extract_selector,omitempty"`
	UserAgent       string   `json:"user_agent,omitempty"`
}

// PageSummary contains summary of a crawled page.
type PageSummary struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	StatusCode int    `json:"status_code"`
	Snippet    string `json:"snippet"`
	LinksFound int    `json:"links_found"`
}

// CrawlResult represents aggregate crawl results.
type CrawlResult struct {
	StartURL     string        `json:"start_url"`
	PagesVisited int           `json:"pages_visited"`
	Pages        []PageSummary `json:"pages"`
	Errors       []string      `json:"errors,omitempty"`
}

const DefaultCrawlerUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 HelperCrawler/1.0"

// CrawlSite executes a polite, bounded multi-page crawl using Colly.
func CrawlSite(ctx context.Context, opts CrawlOptions) (*CrawlResult, error) {
	if opts.StartURL == "" {
		return nil, fmt.Errorf("start_url is required")
	}

	parsedURL, err := url.Parse(opts.StartURL)
	if err != nil {
		return nil, fmt.Errorf("invalid start_url: %w", err)
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("unsupported protocol scheme: '%s' (only http and https are allowed)", scheme)
	}

	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 1
	} else if opts.MaxDepth > 3 {
		opts.MaxDepth = 3 // Hard cap at depth 3
	}

	if opts.MaxPages <= 0 {
		opts.MaxPages = 10
	} else if opts.MaxPages > 50 {
		opts.MaxPages = 50 // Hard safety cap
	}

	if len(opts.AllowedDomains) == 0 && parsedURL.Hostname() != "" {
		opts.AllowedDomains = []string{parsedURL.Hostname()}
	}

	if opts.UserAgent == "" {
		opts.UserAgent = DefaultCrawlerUserAgent
	}

	c := colly.NewCollector(
		colly.MaxDepth(opts.MaxDepth),
		colly.AllowedDomains(opts.AllowedDomains...),
		colly.UserAgent(opts.UserAgent),
		colly.Async(true),
	)

	// Rate limit politeness
	delay := 500 * time.Millisecond
	if opts.RateLimitMs > 0 {
		delay = time.Duration(opts.RateLimitMs) * time.Millisecond
	}

	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       delay,
	})

	result := &CrawlResult{
		StartURL: opts.StartURL,
		Pages:    make([]PageSummary, 0),
	}

	var mu sync.Mutex
	visitedCount := 0

	c.OnHTML("html", func(e *colly.HTMLElement) {
		mu.Lock()
		if visitedCount >= opts.MaxPages {
			mu.Unlock()
			return
		}
		visitedCount++
		currentURL := e.Request.URL.String()
		mu.Unlock()

		title := strings.TrimSpace(e.DOM.Find("title").Text())

		// Extract snippet
		var sel *goquery.Selection
		if opts.ExtractSelector != "" {
			sel = e.DOM.Find(opts.ExtractSelector)
		} else if e.DOM.Find("main").Length() > 0 {
			sel = e.DOM.Find("main")
		} else if e.DOM.Find("article").Length() > 0 {
			sel = e.DOM.Find("article")
		} else {
			sel = e.DOM.Find("body")
		}

		sel.Find("script, style, noscript, svg, nav, footer").Remove()
		text := strings.Join(strings.Fields(sel.Text()), " ")
		if len(text) > 300 {
			text = text[:300] + "..."
		}

		// Count links and queue next
		linksFound := 0
		e.ForEach("a[href]", func(_ int, linkElem *colly.HTMLElement) {
			linksFound++
			mu.Lock()
			shouldVisit := visitedCount < opts.MaxPages
			mu.Unlock()
			if shouldVisit {
				_ = linkElem.Request.Visit(linkElem.Attr("href"))
			}
		})

		summary := PageSummary{
			URL:        currentURL,
			Title:      title,
			StatusCode: e.Response.StatusCode,
			Snippet:    text,
			LinksFound: linksFound,
		}

		mu.Lock()
		result.Pages = append(result.Pages, summary)
		mu.Unlock()
	})

	c.OnError(func(r *colly.Response, err error) {
		mu.Lock()
		defer mu.Unlock()
		reqURL := ""
		if r != nil && r.Request != nil {
			reqURL = r.Request.URL.String()
		}
		result.Errors = append(result.Errors, fmt.Sprintf("error visiting %s: %v", reqURL, err))
	})

	if err := c.Visit(opts.StartURL); err != nil {
		return nil, fmt.Errorf("failed to start crawling: %w", err)
	}

	c.Wait()

	result.PagesVisited = len(result.Pages)
	return result, nil
}
