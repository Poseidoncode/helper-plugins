package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCrawlSite(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/":
			fmt.Fprintf(w, `<html><head><title>Home</title></head><body><h1>Home Page</h1><a href="%s/page1">Page 1</a><a href="%s/page2">Page 2</a></body></html>`, ts.URL, ts.URL)
		case "/page1":
			fmt.Fprintf(w, `<html><head><title>Page 1</title></head><body><h1>Page 1 Content</h1><a href="%s/page2">Page 2</a></body></html>`, ts.URL)
		case "/page2":
			fmt.Fprintf(w, `<html><head><title>Page 2</title></head><body><h1>Page 2 Content</h1></body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	ctx := context.Background()

	res, err := CrawlSite(ctx, CrawlOptions{
		StartURL:    ts.URL + "/",
		MaxDepth:    2,
		MaxPages:    3,
		RateLimitMs: 10,
	})

	if err != nil {
		t.Fatalf("unexpected error crawling: %v", err)
	}

	if res.PagesVisited < 2 {
		t.Errorf("expected at least 2 pages visited, got %d", res.PagesVisited)
	}

	foundHome := false
	for _, p := range res.Pages {
		if strings.Contains(p.Title, "Home") {
			foundHome = true
			break
		}
	}

	if !foundHome {
		t.Errorf("expected to visit Home page in results")
	}
}
