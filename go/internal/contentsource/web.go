package contentsource

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// WebContentSource implements ContentSource for web URLs
type WebContentSource struct {
	name             string
	baseURL          string
	urlList          []string
	urlListFile      string
	refreshInterval  int
	headers          map[string]string
	includePatterns  []*regexp.Regexp
	excludePatterns  []*regexp.Regexp
	maxLinkDepth     int
	client           *http.Client
	contentCache     map[string]*DocumentContent
}

// WebConfig holds configuration for WebContentSource
type WebConfig struct {
	Name            string            `json:"name"`
	BaseURL         string            `json:"base_url"`
	URLList         []string          `json:"url_list"`
	URLListFile     string            `json:"url_list_file"`
	RefreshInterval int               `json:"refresh_interval"`
	Headers         map[string]string `json:"headers"`
	IncludePatterns []string          `json:"include_patterns"`
	ExcludePatterns []string          `json:"exclude_patterns"`
	MaxLinkDepth    int               `json:"max_link_depth"`
	Authentication  map[string]string `json:"authentication"`
}

// NewWebContentSource creates a new web content source
func NewWebContentSource(config map[string]interface{}) (*WebContentSource, error) {
	// Extract configuration
	name, _ := config["name"].(string)
	if name == "" {
		name = "unnamed-web-source"
	}

	baseURL, _ := config["base_url"].(string)
	urlListFile, _ := config["url_list_file"].(string)

	refreshInterval := 86400 // Default: 1 day
	if val, ok := config["refresh_interval"].(float64); ok {
		refreshInterval = int(val)
	} else if val, ok := config["refresh_interval"].(int); ok {
		refreshInterval = val
	}

	maxLinkDepth := 1
	if val, ok := config["max_link_depth"].(float64); ok {
		maxLinkDepth = int(val)
	} else if val, ok := config["max_link_depth"].(int); ok {
		maxLinkDepth = val
	}

	// Extract URL list
	var urlList []string
	if val, ok := config["url_list"].([]interface{}); ok {
		for _, u := range val {
			if s, ok := u.(string); ok {
				urlList = append(urlList, s)
			}
		}
	}

	// Load URLs from file if specified
	if urlListFile != "" {
		if data, err := os.ReadFile(urlListFile); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					urlList = append(urlList, line)
				}
			}
		}
	}

	// Extract headers
	headers := make(map[string]string)
	if val, ok := config["headers"].(map[string]interface{}); ok {
		for k, v := range val {
			if s, ok := v.(string); ok {
				headers[k] = s
			}
		}
	}

	// Extract patterns
	var includePatterns, excludePatterns []*regexp.Regexp
	if val, ok := config["include_patterns"].([]interface{}); ok {
		for _, p := range val {
			if s, ok := p.(string); ok {
				if re, err := regexp.Compile(s); err == nil {
					includePatterns = append(includePatterns, re)
				}
			}
		}
	}
	if val, ok := config["exclude_patterns"].([]interface{}); ok {
		for _, p := range val {
			if s, ok := p.(string); ok {
				if re, err := regexp.Compile(s); err == nil {
					excludePatterns = append(excludePatterns, re)
				}
			}
		}
	}

	// Create HTTP client with connection pooling and keepalive
	transport := &http.Transport{
		MaxIdleConns:        100,              // Increase max idle connections
		MaxIdleConnsPerHost: 10,               // Allow multiple connections per host
		IdleConnTimeout:     90 * time.Second, // Keep connections alive
		DisableKeepAlives:   false,            // Enable HTTP keepalive
		DisableCompression:  false,            // Enable gzip compression
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	// Apply authentication if configured
	if auth, ok := config["authentication"].(map[string]interface{}); ok {
		authType, _ := auth["type"].(string)
		if authType == "basic" {
			username, _ := auth["username"].(string)
			password, _ := auth["password"].(string)
			// Wrap the optimized transport with basic auth
			client.Transport = &basicAuthTransport{
				Username: username,
				Password: password,
				Base:     transport,
			}
		} else if authType == "bearer" {
			token, _ := auth["token"].(string)
			headers["Authorization"] = fmt.Sprintf("Bearer %s", token)
		}
	}

	return &WebContentSource{
		name:            name,
		baseURL:         baseURL,
		urlList:         urlList,
		urlListFile:     urlListFile,
		refreshInterval: refreshInterval,
		headers:         headers,
		includePatterns: includePatterns,
		excludePatterns: excludePatterns,
		maxLinkDepth:    maxLinkDepth,
		client:          client,
		contentCache:    make(map[string]*DocumentContent),
	}, nil
}

// FetchDocument fetches a document from a web URL
func (w *WebContentSource) FetchDocument(sourceID string) (*DocumentContent, error) {
	// sourceID is expected to be a full URL
	targetURL := sourceID

	// Add base URL if it's a relative URL and baseURL is set
	if w.baseURL != "" && !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		parsedBase, err := url.Parse(w.baseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid base URL: %w", err)
		}
		parsedTarget, err := url.Parse(targetURL)
		if err != nil {
			return nil, fmt.Errorf("invalid target URL: %w", err)
		}
		targetURL = parsedBase.ResolveReference(parsedTarget).String()
	}

	// Check cache first
	if cached, ok := w.contentCache[targetURL]; ok {
		return cached, nil
	}

	// Create request
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add headers
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}

	// Fetch content
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching URL %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d for URL %s", resp.StatusCode, targetURL)
	}

	// Read content
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	content := string(body)
	contentType := resp.Header.Get("Content-Type")

	// Get last modified
	lastModified := getLastModified(resp)

	// Create metadata
	metadata := map[string]interface{}{
		"url":           targetURL,
		"content_type":  contentType,
		"last_modified": lastModified,
		"status_code":   resp.StatusCode,
	}

	// Generate content hash
	contentHash := GetContentHash(content)

	doc := &DocumentContent{
		ID:          targetURL,
		Content:     content,
		Metadata:    metadata,
		ContentHash: contentHash,
		DocType:     "html", // Default to HTML for web content
	}

	// Cache the result
	w.contentCache[targetURL] = doc

	return doc, nil
}

// ListDocuments lists available web URLs
func (w *WebContentSource) ListDocuments() ([]DocumentInfo, error) {
	var results []DocumentInfo

	for _, u := range w.urlList {
		// Add base URL if relative and baseURL is set
		fullURL := u
		if w.baseURL != "" && !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			parsedBase, err := url.Parse(w.baseURL)
			if err != nil {
				continue
			}
			parsedTarget, err := url.Parse(u)
			if err != nil {
				continue
			}
			fullURL = parsedBase.ResolveReference(parsedTarget).String()
		}

		metadata := map[string]interface{}{
			"url": fullURL,
		}

		results = append(results, DocumentInfo{
			ID:       fullURL,
			Metadata: metadata,
			DocType:  "html",
		})
	}

	return results, nil
}

// HasChanged checks if web content has changed using HTTP headers
func (w *WebContentSource) HasChanged(sourceID string, lastModified interface{}) (bool, error) {
	targetURL := sourceID

	// Create HEAD request
	req, err := http.NewRequest("HEAD", targetURL, nil)
	if err != nil {
		return true, nil // Assume changed if we can't check
	}

	// Add If-Modified-Since header if we have a last_modified timestamp
	if lastModified != nil {
		var lastModTime float64
		switch v := lastModified.(type) {
		case float64:
			lastModTime = v
		case int64:
			lastModTime = float64(v)
		case int:
			lastModTime = float64(v)
		}

		if lastModTime > 0 {
			t := time.Unix(int64(lastModTime), 0)
			req.Header.Set("If-Modified-Since", t.UTC().Format(http.TimeFormat))
		}
	}

	// Add custom headers
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}

	// Make HEAD request
	resp, err := w.client.Do(req)
	if err != nil {
		return true, nil // Assume changed if request fails
	}
	defer resp.Body.Close()

	// If we get 304 Not Modified, content hasn't changed
	if resp.StatusCode == http.StatusNotModified {
		return false, nil
	}

	// If If-Modified-Since was set and we got 200, content has changed
	if req.Header.Get("If-Modified-Since") != "" && resp.StatusCode == http.StatusOK {
		return true, nil
	}

	// Try to compare Last-Modified headers
	currentLastModified := getLastModified(resp)
	if lastModified == nil || currentLastModified == 0 {
		return true, nil // Can't determine, assume changed
	}

	var lastModTime float64
	switch v := lastModified.(type) {
	case float64:
		lastModTime = v
	case int64:
		lastModTime = float64(v)
	case int:
		lastModTime = float64(v)
	}

	return currentLastModified > lastModTime, nil
}

// getLastModified extracts Last-Modified timestamp from response headers
func getLastModified(resp *http.Response) float64 {
	lastModified := resp.Header.Get("Last-Modified")
	if lastModified == "" {
		return 0
	}

	t, err := http.ParseTime(lastModified)
	if err != nil {
		return 0
	}

	return float64(t.Unix())
}

// ExtractLinks extracts links from HTML content
func (w *WebContentSource) ExtractLinks(content string, sourceURL string) ([]string, error) {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}

	parsedSource, err := url.Parse(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid source URL: %w", err)
	}

	var links []string
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					// Resolve relative URLs
					parsedLink, err := url.Parse(attr.Val)
					if err != nil {
						continue
					}
					absoluteURL := parsedSource.ResolveReference(parsedLink).String()

					// Skip external links (different domain)
					parsedAbs, err := url.Parse(absoluteURL)
					if err != nil {
						continue
					}
					if parsedAbs.Host != parsedSource.Host {
						continue
					}

					// Skip URL fragments
					if absoluteURL == sourceURL || strings.HasPrefix(absoluteURL, sourceURL+"#") {
						continue
					}

					// Check include/exclude patterns
					if w.shouldIncludeURL(absoluteURL) {
						links = append(links, absoluteURL)
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)

	return links, nil
}

// shouldIncludeURL checks if URL should be included based on patterns
func (w *WebContentSource) shouldIncludeURL(targetURL string) bool {
	// Check exclude patterns first
	for _, pattern := range w.excludePatterns {
		if pattern.MatchString(targetURL) {
			return false
		}
	}

	// If no include patterns, include all
	if len(w.includePatterns) == 0 {
		return true
	}

	// Check include patterns
	for _, pattern := range w.includePatterns {
		if pattern.MatchString(targetURL) {
			return true
		}
	}

	return false
}

// basicAuthTransport wraps http.RoundTripper to add basic authentication
type basicAuthTransport struct {
	Username string
	Password string
	Base     http.RoundTripper
}

func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(t.Username, t.Password)
	return t.Base.RoundTrip(req)
}
