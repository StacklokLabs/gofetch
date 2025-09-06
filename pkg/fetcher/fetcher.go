// Package fetcher provides HTTP content fetching and processing functionality.
package fetcher

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/stackloklabs/gofetch/pkg/observability"
	"github.com/stackloklabs/gofetch/pkg/processor"
	"github.com/stackloklabs/gofetch/pkg/robots"
)

// HTTPFetcher handles HTTP requests and content retrieval
type HTTPFetcher struct {
	httpClient    *http.Client
	robotsChecker *robots.Checker
	processor     *processor.ContentProcessor
	userAgent     string
	metrics       *observability.Metrics
}

// NewHTTPFetcher creates a new HTTP fetcher instance
func NewHTTPFetcher(
	httpClient *http.Client,
	robotsChecker *robots.Checker,
	contentProcessor *processor.ContentProcessor,
	userAgent string,
	metrics *observability.Metrics,
) *HTTPFetcher {
	return &HTTPFetcher{
		httpClient:    httpClient,
		robotsChecker: robotsChecker,
		processor:     contentProcessor,
		userAgent:     userAgent,
		metrics:       metrics,
	}
}

// FetchRequest holds the parameters for a fetch request
type FetchRequest struct {
	URL        string
	MaxLength  *int
	StartIndex *int
	Raw        bool
}

// FetchURL retrieves and processes content from the specified URL
func (f *HTTPFetcher) FetchURL(req *FetchRequest) (string, error) {
	ctx := context.Background()
	log.Printf("Fetching URL: %s", req.URL)

	// Check robots.txt
	if f.metrics != nil {
		f.metrics.RecordRobotsCheck(ctx, req.URL, "checking")
	}
	
	if !f.robotsChecker.IsAllowed(req.URL) {
		log.Printf("Access denied by robots.txt for URL: %s", req.URL)
		if f.metrics != nil {
			f.metrics.RecordRobotsCheck(ctx, req.URL, "disallowed")
			f.metrics.RecordRobotsBlocked(ctx, req.URL)
		}
		return "", fmt.Errorf("access to %s is disallowed by robots.txt", req.URL)
	}
	
	if f.metrics != nil {
		f.metrics.RecordRobotsCheck(ctx, req.URL, "allowed")
	}

	// Fetch the content
	content, err := f.fetchURL(req.URL, req.Raw)
	if err != nil {
		return "", err
	}

	// Apply formatting
	formattedContent := f.processor.FormatContent(content, req.StartIndex, req.MaxLength)

	log.Printf("Fetch completed successfully for %s, returning %d characters", req.URL, len(formattedContent))
	return formattedContent, nil
}

// fetchURL retrieves content from the specified URL
func (f *HTTPFetcher) fetchURL(url string, raw bool) (string, error) {
	ctx := context.Background()
	
	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to create HTTP request for %s: %v", url, err)
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	// Make HTTP request
	resp, err := f.httpClient.Do(req)
	if err != nil {
		log.Printf("HTTP request failed for %s: %v", url, err)
		return "", fmt.Errorf("failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("HTTP %d response from %s (Content-Type: %s)", resp.StatusCode, url, resp.Header.Get("Content-Type"))

	// Check status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("Non-200 status code %d for %s: %s", resp.StatusCode, url, resp.Status)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body from %s: %v", url, err)
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	log.Printf("Successfully fetched %d bytes from %s", len(body), url)

	content := string(body)
	originalSize := len(body)

	// Process HTML if not raw mode
	if !raw && strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		startTime := time.Now()
		processedContent := f.processor.ProcessHTML(content)
		processingDuration := time.Since(startTime)
		
		if f.metrics != nil {
			f.metrics.RecordHTMLToMarkdownConversion(ctx, true)
			f.metrics.RecordContentProcessing(ctx, "text/html", processingDuration)
			f.metrics.RecordContentSizeRatio(ctx, originalSize, len(processedContent))
		}
		
		content = processedContent
	}

	return content, nil
}
