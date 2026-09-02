package browser

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

// RenderOptions defines options for Chromedp browser rendering.
type RenderOptions struct {
	URL               string        `json:"url"`
	WaitSelector      string        `json:"wait_selector,omitempty"`
	WaitSeconds       int           `json:"wait_seconds,omitempty"`
	Timeout           time.Duration `json:"timeout,omitempty"`
	ExtractSelector   string        `json:"extract_selector,omitempty"`
	ScreenshotPath    string        `json:"screenshot_path,omitempty"`
	ConvertToMarkdown bool          `json:"convert_markdown,omitempty"`
	Headless          bool          `json:"headless,omitempty"`
	UserAgent         string        `json:"user_agent,omitempty"`
}

// RenderResult represents the output of dynamic page rendering.
type RenderResult struct {
	URL            string `json:"url"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	ScreenshotPath string `json:"screenshot_path,omitempty"`
	RenderedHTML   string `json:"rendered_html,omitempty"`
}

const DefaultBrowserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 HelperBot/1.0"

// ValidateBrowserURL ensures the URL only uses HTTP or HTTPS.
func ValidateBrowserURL(rawURL string) error {
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

// ResolveSafeScreenshotPath ensures the screenshot path does not escape into sensitive system directories.
func ResolveSafeScreenshotPath(userPath string) (string, error) {
	if userPath == "" {
		return "", nil
	}

	cleanPath := filepath.Clean(userPath)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve screenshot path: %w", err)
	}

	// Disallow writing directly to sensitive root system directories
	forbiddenPrefixes := []string{"/etc", "/bin", "/sbin", "/usr", "/sys", "/proc", "/dev"}
	for _, pref := range forbiddenPrefixes {
		if strings.HasPrefix(absPath, pref) {
			return "", fmt.Errorf("screenshot path cannot be in system directory: %s", pref)
		}
	}

	return absPath, nil
}

// RenderDynamic launches Headless Chrome via chromedp, waits for JS execution, and extracts DOM.
func RenderDynamic(ctx context.Context, opts RenderOptions) (*RenderResult, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("url is required")
	}

	if err := ValidateBrowserURL(opts.URL); err != nil {
		return nil, err
	}

	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.UserAgent == "" {
		opts.UserAgent = DefaultBrowserUserAgent
	}

	var safeScreenshotPath string
	if opts.ScreenshotPath != "" {
		p, err := ResolveSafeScreenshotPath(opts.ScreenshotPath)
		if err != nil {
			return nil, err
		}
		safeScreenshotPath = p
	}

	// Prepare Chromedp allocator options
	execOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.UserAgent(opts.UserAgent),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, execOpts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	timeoutCtx, cancelTimeout := context.WithTimeout(taskCtx, opts.Timeout)
	defer cancelTimeout()

	var pageTitle string
	var renderedHTML string
	var screenshotBuf []byte

	var tasks chromedp.Tasks
	tasks = append(tasks, chromedp.Navigate(opts.URL))

	if opts.WaitSelector != "" {
		tasks = append(tasks, chromedp.WaitVisible(opts.WaitSelector, chromedp.ByQuery))
	}

	if opts.WaitSeconds > 0 {
		tasks = append(tasks, chromedp.Sleep(time.Duration(opts.WaitSeconds)*time.Second))
	}

	tasks = append(tasks,
		chromedp.Title(&pageTitle),
		chromedp.OuterHTML("html", &renderedHTML, chromedp.ByQuery),
	)

	if safeScreenshotPath != "" {
		tasks = append(tasks, chromedp.FullScreenshot(&screenshotBuf, 90))
	}

	err := chromedp.Run(timeoutCtx, tasks)
	if err != nil {
		return nil, fmt.Errorf("chromedp execution failed: %w", err)
	}

	result := &RenderResult{
		URL:   opts.URL,
		Title: strings.TrimSpace(pageTitle),
	}

	// Handle Screenshot save safely
	if safeScreenshotPath != "" && len(screenshotBuf) > 0 {
		if err := os.MkdirAll(filepath.Dir(safeScreenshotPath), 0755); err == nil {
			if err := os.WriteFile(safeScreenshotPath, screenshotBuf, 0644); err == nil {
				result.ScreenshotPath = safeScreenshotPath
			} else {
				return nil, fmt.Errorf("failed to save screenshot: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to create screenshot directory: %w", err)
		}
	}

	// Parse rendered HTML with Goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(renderedHTML))
	if err != nil {
		result.Content = renderedHTML
		return result, nil
	}

	var targetSelection *goquery.Selection
	if opts.ExtractSelector != "" {
		targetSelection = doc.Find(opts.ExtractSelector)
	} else {
		if doc.Find("main").Length() > 0 {
			targetSelection = doc.Find("main")
		} else if doc.Find("article").Length() > 0 {
			targetSelection = doc.Find("article")
		} else {
			targetSelection = doc.Find("body")
		}
	}

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
