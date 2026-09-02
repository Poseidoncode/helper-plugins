package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestScrapeStatic(t *testing.T) {
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>Test Page Title</title>
    <meta name="description" content="A test page description">
</head>
<body>
    <header><h1>Header Content</h1></header>
    <main>
        <h2>Main Article</h2>
        <p>This is a <strong>test paragraph</strong> with <a href="/subpage">internal link</a> and <a href="https://example.com/ext">external link</a>.</p>
    </main>
    <footer>Footer notes</footer>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	ctx := context.Background()

	t.Run("Scrape with Markdown conversion", func(t *testing.T) {
		res, err := ScrapeStatic(ctx, ScrapeOptions{
			URL:               server.URL,
			ConvertToMarkdown: true,
			ExtractLinks:      true,
			Timeout:           5 * time.Second,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.Title != "Test Page Title" {
			t.Errorf("expected title 'Test Page Title', got '%s'", res.Title)
		}

		if res.Metadata["description"] != "A test page description" {
			t.Errorf("expected meta description 'A test page description', got '%s'", res.Metadata["description"])
		}

		if len(res.Links) != 2 {
			t.Errorf("expected 2 links, got %d: %v", len(res.Links), res.Links)
		}

		if res.StatusCode != 200 {
			t.Errorf("expected status code 200, got %d", res.StatusCode)
		}
	})

	t.Run("Scrape with CSS Selector", func(t *testing.T) {
		res, err := ScrapeStatic(ctx, ScrapeOptions{
			URL:               server.URL,
			Selector:          "main h2",
			ConvertToMarkdown: false,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.Content != "Main Article" {
			t.Errorf("expected content 'Main Article', got '%s'", res.Content)
		}
	})
}
