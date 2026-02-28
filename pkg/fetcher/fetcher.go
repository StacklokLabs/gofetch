// Package fetcher provides HTTP content fetching and processing functionality.
package fetcher

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/stackloklabs/gofetch/pkg/processor"
	"github.com/stackloklabs/gofetch/pkg/robots"
)

// HTTPFetcher handles HTTP requests and content retrieval
type HTTPFetcher struct {
	httpClient    *http.Client
	robotsChecker *robots.Checker
	processor     *processor.ContentProcessor
	userAgent     string
}

// NewHTTPFetcher creates a new HTTP fetcher instance
func NewHTTPFetcher(
	httpClient *http.Client,
	robotsChecker *robots.Checker,
	contentProcessor *processor.ContentProcessor,
	userAgent string,
) *HTTPFetcher {
	return &HTTPFetcher{
		httpClient:    httpClient,
		robotsChecker: robotsChecker,
		processor:     contentProcessor,
		userAgent:     userAgent,
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
	log.Printf("Fetching URL: %s", sanitizeLogValue(req.URL))

	// Check robots.txt
	if !f.robotsChecker.IsAllowed(req.URL) {
		log.Printf("Access denied by robots.txt for URL: %s", sanitizeLogValue(req.URL))
		return "", fmt.Errorf("access to %s is disallowed by robots.txt", req.URL)
	}

	// Fetch the content
	content, err := f.fetchURL(req.URL, req.Raw)
	if err != nil {
		return "", err
	}

	// Apply formatting
	formattedContent := f.processor.FormatContent(content, req.StartIndex, req.MaxLength)

	log.Printf("Fetch completed successfully for %s, returning %d characters", sanitizeLogValue(req.URL), len(formattedContent))
	return formattedContent, nil
}

// sanitizeLogValue removes newlines and carriage returns to prevent log injection.
func sanitizeLogValue(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// fetchURL retrieves content from the specified URL
func (f *HTTPFetcher) fetchURL(url string, raw bool) (string, error) {
	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to create HTTP request for %s: %v", sanitizeLogValue(url), err)
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	// Make HTTP request
	resp, err := f.httpClient.Do(req) //nolint:gosec // This is a fetch server; fetching user-provided URLs is its core purpose
	if err != nil {
		log.Printf("HTTP request failed for %s: %v", sanitizeLogValue(url), err)
		return "", fmt.Errorf("failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	//nolint:gosec // URL sanitized by sanitizeLogValue; gosec can't track custom sanitizers
	log.Printf("HTTP %d response from %s (Content-Type: %s)",
		resp.StatusCode, sanitizeLogValue(url), resp.Header.Get("Content-Type"))

	// Check status code
	if resp.StatusCode != http.StatusOK {
		//nolint:gosec // URL sanitized by sanitizeLogValue; gosec can't track custom sanitizers
		log.Printf("Non-200 status code %d for %s: %s",
			resp.StatusCode, sanitizeLogValue(url), resp.Status)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body from %s: %v", sanitizeLogValue(url), err)
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	log.Printf("Successfully fetched %d bytes from %s", len(body), sanitizeLogValue(url))

	content := string(body)

	// Process HTML if not raw mode
	if !raw && strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		content = f.processor.ProcessHTML(content)
	}

	return content, nil
}
