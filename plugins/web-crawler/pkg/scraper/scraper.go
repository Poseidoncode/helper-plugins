package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// ScrapeOptions defines parameters for static scraping.
type ScrapeOptions struct {
	URL               string        `json:"url"`
	Selector          string        `json:"selector,omitempty"`
	ExtractLinks      bool          `json:"extract_links,omitempty"`
	ConvertToMarkdown bool          `json:"convert_markdown,omitempty"`
	Timeout           time.Duration `json:"timeout,omitempty"`
	UserAgent         string        `json:"user_agent,omitempty"`
}

// ScrapeResult represents the scraped webpage output.
type ScrapeResult struct {
	URL        string            `json:"url"`
	Title      string            `json:"title"`
	StatusCode int               `json:"status_code"`
	Content    string            `json:"content"`
	Links      []string          `json:"links,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Error      string            `json:"error,omitempty"`
}

// DefaultUserAgent used if none is specified.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 HelperBot/1.0"

// ValidateScheme ensures the URL only uses HTTP or HTTPS to prevent SSRF and local file exposure.
func ValidateScheme(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported protocol scheme: '%s' (only http and https are allowed)", scheme)
	}
	return nil
}

// ScrapeStatic fetches a static webpage using Colly and parses DOM using Goquery.
func ScrapeStatic(ctx context.Context, opts ScrapeOptions) (*ScrapeResult, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("url is required")
	}

	if err := ValidateScheme(opts.URL); err != nil {
		return nil, err
	}

	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}
	if opts.UserAgent == "" {
		opts.UserAgent = DefaultUserAgent
	}

	c := colly.NewCollector(
		colly.UserAgent(opts.UserAgent),
		colly.Async(false),
	)
	c.SetRequestTimeout(opts.Timeout)

	result := &ScrapeResult{
		URL:      opts.URL,
		Metadata: make(map[string]string),
	}

	var rawHTML string

	c.OnResponse(func(r *colly.Response) {
		result.StatusCode = r.StatusCode
		rawHTML = string(r.Body)
	})

	c.OnError(func(r *colly.Response, err error) {
		if r != nil {
			result.StatusCode = r.StatusCode
		}
		result.Error = err.Error()
	})

	err := c.Visit(opts.URL)
	if err != nil && result.Error == "" {
		return nil, fmt.Errorf("failed to visit url: %w", err)
	}

	if result.StatusCode == 0 && result.Error != "" {
		return nil, fmt.Errorf("scraping error: %s", result.Error)
	}

	// Parse with Goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf("failed to parse html: %w", err)
	}

	// Extract Title
	result.Title = strings.TrimSpace(doc.Find("title").Text())

	// Extract Meta tags
	doc.Find("meta").Each(func(_ int, s *goquery.Selection) {
		name, exists := s.Attr("name")
		if !exists {
			name, exists = s.Attr("property")
		}
		if exists && name != "" {
			content, exists := s.Attr("content")
			if exists && content != "" {
				result.Metadata[strings.ToLower(name)] = content
			}
		}
	})

	// Extract Links if requested
	if opts.ExtractLinks {
		linkMap := make(map[string]bool)
		doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if exists {
				href = strings.TrimSpace(href)
				if href != "" && !strings.HasPrefix(href, "javascript:") && !strings.HasPrefix(href, "#") {
					if !linkMap[href] {
						linkMap[href] = true
						result.Links = append(result.Links, href)
					}
				}
			}
		})
	}

	// Target content container
	var targetSelection *goquery.Selection
	if opts.Selector != "" {
		targetSelection = doc.Find(opts.Selector)
	} else {
		if doc.Find("main").Length() > 0 {
			targetSelection = doc.Find("main")
		} else if doc.Find("article").Length() > 0 {
			targetSelection = doc.Find("article")
		} else {
			targetSelection = doc.Find("body")
		}
	}

	// Clean unwanted elements
	targetSelection.Find("script, style, noscript, svg, iframe").Remove()

	targetHTML, _ := targetSelection.Html()

	if opts.ConvertToMarkdown {
		mdBytes, err := md.ConvertString(targetHTML)
		if err == nil {
			result.Content = strings.TrimSpace(string(mdBytes))
		} else {
			result.Content = strings.TrimSpace(targetSelection.Text())
		}
	} else {
		result.Content = strings.TrimSpace(targetSelection.Text())
	}

	return result, nil
}
