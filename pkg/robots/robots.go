// Package robots provides robots.txt validation functionality.
package robots

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Checker handles robots.txt validation for web crawling
type Checker struct {
	userAgent    string
	ignoreRobots bool
	httpClient   *http.Client
}

// NewChecker creates a new robots.txt checker
func NewChecker(userAgent string, ignoreRobots bool, httpClient *http.Client) *Checker {
	return &Checker{
		userAgent:    userAgent,
		ignoreRobots: ignoreRobots,
		httpClient:   httpClient,
	}
}

// IsAllowed checks if the URL can be accessed according to robots.txt
func (c *Checker) IsAllowed(targetURL string) bool {
	if c.ignoreRobots {
		return true
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	robotsContent, err := c.fetchRobotsContent(parsedURL)
	if err != nil {
		// If we can't fetch robots.txt, allow access
		return true
	}

	return c.parseRobotsRules(robotsContent, parsedURL.Path)
}

// fetchRobotsContent retrieves the robots.txt file for a given URL
func (c *Checker) fetchRobotsContent(parsedURL *url.URL) (string, error) {
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsedURL.Scheme, parsedURL.Host)

	req, err := http.NewRequest("GET", robotsURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req) //nolint:gosec // Fetching robots.txt for user-provided URLs is expected behavior
	if err != nil || resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch robots.txt")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// parseRobotsRules parses robots.txt content and checks if access is allowed
func (c *Checker) parseRobotsRules(robotsContent, targetPath string) bool {
	userAgentPattern := regexp.MustCompile(`(?i)^User-agent:\s*(.+)$`)
	disallowPattern := regexp.MustCompile(`(?i)^Disallow:\s*(.*)$`) // Allow empty disallow rules

	lines := strings.Split(robotsContent, "\n")
	var currentUserAgents []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if userAgentMatch := userAgentPattern.FindStringSubmatch(line); userAgentMatch != nil {
			userAgent := strings.TrimSpace(userAgentMatch[1])
			if userAgent == "*" || strings.Contains(c.userAgent, userAgent) {
				currentUserAgents = append(currentUserAgents, userAgent)
			}
		} else if disallowMatch := disallowPattern.FindStringSubmatch(line); disallowMatch != nil && len(currentUserAgents) > 0 {
			disallowPath := strings.TrimSpace(disallowMatch[1])
			// Empty disallow means allow everything for this user agent
			if disallowPath == "" {
				continue
			}
			if disallowPath == "/" || strings.HasPrefix(targetPath, disallowPath) {
				return false
			}
		}
	}

	return true
}
