package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/Poseidoncode/helper-plugins/plugins/web-crawler/pkg/browser"
	"github.com/Poseidoncode/helper-plugins/plugins/web-crawler/pkg/crawler"
	"github.com/Poseidoncode/helper-plugins/plugins/web-crawler/pkg/scraper"
)

// JSONRPCRequest represents an incoming JSON-RPC request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC response.
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError defines a standard JSON-RPC error.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// Server is the MCP stdio server with thread-safe output.
type Server struct {
	reader io.Reader
	writer io.Writer
	mu     sync.Mutex // Protects writer from interleaved writes
}

// NewServer creates a new MCP server.
func NewServer(r io.Reader, w io.Writer) *Server {
	return &Server{
		reader: r,
		writer: w,
	}
}

// Run starts the stdio JSON-RPC loop.
func (s *Server) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(s.reader)
	// Support large payload lines (up to 10MB)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, fmt.Sprintf("Parse error: %v", err))
			continue
		}

		s.handleRequest(ctx, req)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (s *Server) handleRequest(ctx context.Context, req JSONRPCRequest) {
	// Notifications (no id)
	if req.ID == nil {
		if req.Method == "notifications/initialized" || req.Method == "initialized" {
			return
		}
		return
	}

	switch req.Method {
	case "initialize":
		s.sendResult(req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "web-crawler",
				"version": "1.0.0",
			},
		})

	case "ping":
		s.sendResult(req.ID, map[string]interface{}{})

	case "tools/list":
		s.sendResult(req.ID, map[string]interface{}{
			"tools": s.listTools(),
		})

	case "tools/call":
		s.handleToolCall(ctx, req.ID, req.Params)

	default:
		s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) listTools() []Tool {
	return []Tool{
		{
			Name:        "scrape_static",
			Description: "Fast static webpage scraping using Colly and Goquery. Best for documentation, blogs, news, and static HTML. Converts HTML to clean Markdown and extracts structured metadata and links.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The target HTTP/HTTPS URL to scrape.",
					},
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "Optional CSS selector to scope content extraction (e.g. 'article.main', '#content').",
					},
					"extract_links": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to extract all hyperlinks on the page (default: false).",
					},
					"convert_markdown": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to convert HTML to Markdown (default: true).",
					},
					"timeout_seconds": map[string]interface{}{
						"type":        "integer",
						"description": "Timeout in seconds (default: 15, max: 120).",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "scrape_dynamic",
			Description: "Dynamic JavaScript/SPA page rendering using Headless Chrome (Chromedp). Supports waiting for DOM selectors, sleep intervals, full page screenshots, and Markdown extraction.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The target URL to render in Headless Chrome.",
					},
					"wait_selector": map[string]interface{}{
						"type":        "string",
						"description": "Optional CSS selector to wait until visible before extracting (e.g. '.data-loaded').",
					},
					"wait_seconds": map[string]interface{}{
						"type":        "integer",
						"description": "Optional additional seconds to sleep after navigation for JS hydration.",
					},
					"extract_selector": map[string]interface{}{
						"type":        "string",
						"description": "Optional CSS selector to extract content from.",
					},
					"screenshot_path": map[string]interface{}{
						"type":        "string",
						"description": "Optional local file path to save a full-page PNG screenshot.",
					},
					"convert_markdown": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to convert rendered HTML to Markdown (default: true).",
					},
					"timeout_seconds": map[string]interface{}{
						"type":        "integer",
						"description": "Overall timeout in seconds (default: 30, max: 120).",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "crawl_site",
			Description: "Bounded multi-page crawler using Colly with depth control, domain scoping, and polite rate limiting. Collects page summaries and links across pages.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"start_url": map[string]interface{}{
						"type":        "string",
						"description": "The entry URL to start crawling from.",
					},
					"allowed_domains": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"description": "Optional list of allowed domain hostnames to restrict crawling.",
					},
					"max_depth": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum crawl depth from start_url (default: 1, max: 3).",
					},
					"max_pages": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of pages to visit (default: 10, max: 50).",
					},
					"rate_limit_ms": map[string]interface{}{
						"type":        "integer",
						"description": "Politeness delay in milliseconds between requests (default: 500ms).",
					},
					"extract_selector": map[string]interface{}{
						"type":        "string",
						"description": "Optional CSS selector to extract page snippet from.",
					},
				},
				"required": []string{"start_url"},
			},
		},
	}
}

func (s *Server) handleToolCall(ctx context.Context, id interface{}, rawParams json.RawMessage) {
	var callParams struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(rawParams, &callParams); err != nil {
		s.sendError(id, -32602, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	if callParams.Arguments == nil {
		callParams.Arguments = make(map[string]interface{})
	}

	switch callParams.Name {
	case "scrape_static":
		s.handleScrapeStatic(ctx, id, callParams.Arguments)
	case "scrape_dynamic":
		s.handleScrapeDynamic(ctx, id, callParams.Arguments)
	case "crawl_site":
		s.handleCrawlSite(ctx, id, callParams.Arguments)
	default:
		s.sendError(id, -32601, fmt.Sprintf("Tool not found: %s", callParams.Name))
	}
}

func (s *Server) handleScrapeStatic(ctx context.Context, id interface{}, args map[string]interface{}) {
	rawURL, _ := args["url"].(string)
	if rawURL == "" {
		s.sendToolError(id, "url parameter is required")
		return
	}

	selector, _ := args["selector"].(string)
	extractLinks, _ := args["extract_links"].(bool)
	convertMarkdown := true
	if cm, ok := args["convert_markdown"].(bool); ok {
		convertMarkdown = cm
	}

	timeoutSec := 15
	if t, ok := args["timeout_seconds"].(float64); ok && t > 0 {
		timeoutSec = int(t)
		if timeoutSec > 120 {
			timeoutSec = 120
		}
	}

	opts := scraper.ScrapeOptions{
		URL:               rawURL,
		Selector:          selector,
		ExtractLinks:      extractLinks,
		ConvertToMarkdown: convertMarkdown,
		Timeout:           time.Duration(timeoutSec) * time.Second,
	}

	res, err := scraper.ScrapeStatic(ctx, opts)
	if err != nil {
		s.sendToolError(id, fmt.Sprintf("scrape_static failed: %v", err))
		return
	}

	jsonBytes, _ := json.MarshalIndent(res, "", "  ")
	s.sendToolSuccess(id, string(jsonBytes))
}

func (s *Server) handleScrapeDynamic(ctx context.Context, id interface{}, args map[string]interface{}) {
	rawURL, _ := args["url"].(string)
	if rawURL == "" {
		s.sendToolError(id, "url parameter is required")
		return
	}

	waitSelector, _ := args["wait_selector"].(string)
	waitSeconds := 0
	if ws, ok := args["wait_seconds"].(float64); ok {
		waitSeconds = int(ws)
	}
	extractSelector, _ := args["extract_selector"].(string)
	screenshotPath, _ := args["screenshot_path"].(string)

	convertMarkdown := true
	if cm, ok := args["convert_markdown"].(bool); ok {
		convertMarkdown = cm
	}

	timeoutSec := 30
	if t, ok := args["timeout_seconds"].(float64); ok && t > 0 {
		timeoutSec = int(t)
		if timeoutSec > 120 {
			timeoutSec = 120
		}
	}

	opts := browser.RenderOptions{
		URL:               rawURL,
		WaitSelector:      waitSelector,
		WaitSeconds:       waitSeconds,
		Timeout:           time.Duration(timeoutSec) * time.Second,
		ExtractSelector:   extractSelector,
		ScreenshotPath:    screenshotPath,
		ConvertToMarkdown: convertMarkdown,
		Headless:          true,
	}

	res, err := browser.RenderDynamic(ctx, opts)
	if err != nil {
		s.sendToolError(id, fmt.Sprintf("scrape_dynamic failed: %v", err))
		return
	}

	jsonBytes, _ := json.MarshalIndent(res, "", "  ")
	s.sendToolSuccess(id, string(jsonBytes))
}

func (s *Server) handleCrawlSite(ctx context.Context, id interface{}, args map[string]interface{}) {
	startURL, _ := args["start_url"].(string)
	if startURL == "" {
		s.sendToolError(id, "start_url parameter is required")
		return
	}

	var allowedDomains []string
	if rawDomains, ok := args["allowed_domains"].([]interface{}); ok {
		for _, d := range rawDomains {
			if str, ok := d.(string); ok && str != "" {
				allowedDomains = append(allowedDomains, str)
			}
		}
	}

	maxDepth := 1
	if md, ok := args["max_depth"].(float64); ok && md > 0 {
		maxDepth = int(md)
		if maxDepth > 3 {
			maxDepth = 3
		}
	}

	maxPages := 10
	if mp, ok := args["max_pages"].(float64); ok && mp > 0 {
		maxPages = int(mp)
		if maxPages > 50 {
			maxPages = 50
		}
	}

	rateLimitMs := 500
	if rl, ok := args["rate_limit_ms"].(float64); ok && rl >= 0 {
		rateLimitMs = int(rl)
	}

	extractSelector, _ := args["extract_selector"].(string)

	opts := crawler.CrawlOptions{
		StartURL:        startURL,
		AllowedDomains:  allowedDomains,
		MaxDepth:        maxDepth,
		MaxPages:        maxPages,
		RateLimitMs:     rateLimitMs,
		ExtractSelector: extractSelector,
	}

	res, err := crawler.CrawlSite(ctx, opts)
	if err != nil {
		s.sendToolError(id, fmt.Sprintf("crawl_site failed: %v", err))
		return
	}

	jsonBytes, _ := json.MarshalIndent(res, "", "  ")
	s.sendToolSuccess(id, string(jsonBytes))
}

func (s *Server) sendToolSuccess(id interface{}, text string) {
	s.sendResult(id, map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": text,
			},
		},
		"isError": false,
	})
}

func (s *Server) sendToolError(id interface{}, errMsg string) {
	s.sendResult(id, map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": errMsg,
			},
		},
		"isError": true,
	})
}

func (s *Server) sendResult(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.writeJSON(resp)
}

func (s *Server) sendError(id interface{}, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
	s.writeJSON(resp)
}

func (s *Server) writeJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "json marshal error: %v\n", err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.writer.Write(append(data, '\n'))
}
