package fetcher

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackloklabs/gofetch/pkg/processor"
	"github.com/stackloklabs/gofetch/pkg/robots"
)

// createMockServer creates a test HTTP server with various endpoints
func createMockServer() *httptest.Server {
	mux := http.NewServeMux()

	// HTML content endpoint
	mux.HandleFunc("/html", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>Test Page</h1><p>This is a test page.</p></body></html>`))
	})

	// JSON content endpoint
	mux.HandleFunc("/json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Hello, World!", "status": "ok"}`))
	})

	// robots.txt endpoint
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		robotsContent := `User-agent: *
Disallow: /private/
Disallow: /admin/

User-agent: TestBot
Disallow: /blocked/`
		w.Write([]byte(robotsContent))
	})

	// Error endpoint
	mux.HandleFunc("/error", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	})

	return httptest.NewServer(mux)
}

func createTestFetcher() *HTTPFetcher {
	client := &http.Client{Timeout: 5 * time.Second}
	robotsChecker := robots.NewChecker("TestBot/1.0", false, client)
	contentProcessor := processor.NewContentProcessor()

	return NewHTTPFetcher(client, robotsChecker, contentProcessor, "TestBot/1.0")
}

func TestNewHTTPFetcher(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	robotsChecker := robots.NewChecker("TestBot/1.0", false, client)
	contentProcessor := processor.NewContentProcessor()
	userAgent := "TestBot/1.0"

	fetcher := NewHTTPFetcher(client, robotsChecker, contentProcessor, userAgent)

	if fetcher.httpClient != client {
		t.Error("expected httpClient to be set correctly")
	}

	if fetcher.robotsChecker != robotsChecker {
		t.Error("expected robotsChecker to be set correctly")
	}

	if fetcher.processor != contentProcessor {
		t.Error("expected processor to be set correctly")
	}

	if fetcher.userAgent != userAgent {
		t.Errorf("expected userAgent %q, got %q", userAgent, fetcher.userAgent)
	}
}

func TestFetchURL(t *testing.T) {
	server := createMockServer()
	defer server.Close()

	fetcher := createTestFetcher()

	tests := []struct {
		name        string
		request     *FetchRequest
		expectError bool
		expectedLen int // approximate length check
	}{
		{
			name: "successful HTML fetch",
			request: &FetchRequest{
				URL: server.URL + "/html",
				Raw: false,
			},
			expectError: false,
			expectedLen: 10, // Should have some content after markdown conversion
		},
		{
			name: "successful JSON fetch",
			request: &FetchRequest{
				URL: server.URL + "/json",
				Raw: true,
			},
			expectError: false,
			expectedLen: 30, // JSON content length
		},
		{
			name: "server error",
			request: &FetchRequest{
				URL: server.URL + "/error",
				Raw: false,
			},
			expectError: true,
			expectedLen: 0,
		},
		{
			name: "blocked by robots.txt",
			request: &FetchRequest{
				URL: server.URL + "/blocked/page",
				Raw: false,
			},
			expectError: true,
			expectedLen: 0,
		},
		{
			name: "fetch with formatting",
			request: &FetchRequest{
				URL:       server.URL + "/json",
				Raw:       true,
				MaxLength: intPtr(20),
			},
			expectError: false,
			expectedLen: 20, // Should be truncated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fetcher.FetchURL(tt.request)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if len(result) < tt.expectedLen {
					t.Errorf("expected result length >= %d, got %d", tt.expectedLen, len(result))
				}
			}
		})
	}
}

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}
