package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func testWikipedia(userAgent string, additionalHeaders map[string]string) {
	url := "https://en.wikipedia.org/wiki/Medicine"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("❌ Error creating request: %v\n", err)
		return
	}

	// Set User-Agent
	req.Header.Set("User-Agent", userAgent)

	// Set additional headers
	for k, v := range additionalHeaders {
		req.Header.Set(k, v)
	}

	fmt.Printf("\n🧪 Testing with User-Agent: %s\n", userAgent)
	fmt.Printf("   Additional headers: %v\n", additionalHeaders)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("   ❌ Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		fmt.Printf("   ✅ SUCCESS - Status: %d, Content length: %d bytes\n", resp.StatusCode, len(body))
	} else {
		fmt.Printf("   ❌ FAILED - Status: %d\n", resp.StatusCode)
	}
}

func main() {
	fmt.Println("Testing Wikipedia Header Requirements")
	fmt.Println("======================================")

	// Test 1: Our current User-Agent
	testWikipedia("Go-Doc-Go/1.0 (test configuration; educational project)", nil)

	// Test 2: Browser-like User-Agent
	testWikipedia("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", nil)

	// Test 3: Python requests style
	testWikipedia("python-requests/2.31.0", nil)

	// Test 4: More complete bot identification
	testWikipedia("Go-Doc-Go/1.0 (+https://github.com/yourusername/go-doc-go; educational project)", nil)

	// Test 5: Browser-like with Accept headers
	testWikipedia(
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		map[string]string{
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.9",
		},
	)

	// Test 6: Curl-style
	testWikipedia("curl/8.0.0", nil)

	// Test 7: Wget-style
	testWikipedia("Wget/1.21.3", nil)

	fmt.Println("\n✅ Testing complete!")
}