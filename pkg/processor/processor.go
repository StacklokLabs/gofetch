// Package processor handles content processing and formatting.
package processor

import (
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/go-shiori/go-readability"
	"golang.org/x/net/html"
)

// ContentProcessor handles HTML processing and content formatting
type ContentProcessor struct {
	// No longer need to store a converter instance in v2
}

// NewContentProcessor creates a new content processor instance
func NewContentProcessor() *ContentProcessor {
	return &ContentProcessor{}
}

// ProcessHTML converts HTML content to readable markdown
func (p *ContentProcessor) ProcessHTML(htmlContent string) string {
	// Parse HTML document
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	// Extract readable content using readability
	article, err := readability.FromDocument(doc, nil)
	if err == nil && article.Content != "" {
		htmlContent = article.Content
	}

	// Convert to markdown using the new v2 API
	markdown, err := htmltomarkdown.ConvertString(htmlContent)
	if err != nil {
		return htmlContent
	}

	return markdown
}

// FormatContent applies pagination and truncation to content
func (*ContentProcessor) FormatContent(content string, startIndex, maxLength *int) string {
	// Apply start index offset
	start := 0
	if startIndex != nil {
		start = *startIndex
	}

	if start > len(content) {
		start = len(content)
	}

	content = content[start:]

	// Apply length limit
	if maxLength != nil && len(content) > *maxLength {
		content = content[:*maxLength]
		content += "\n\n[Content truncated. Use start_index to get more content.]"
	}

	return content
}
