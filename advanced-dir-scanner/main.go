package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ScanConfig holds the configuration for a directory scan
type ScanConfig struct {
	TargetURL   string
	Wordlist    []string
	Concurrency int
	Recursive   bool
	TimeoutMs   int
}

// ScanResult represents the result of scanning a single path
type ScanResult struct {
	Path       string `json:"path"`
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
	Size       int64  `json:"size"`
}

// Global state for simplicity in this example
var (
	scanResults  []ScanResult
	resultsMutex sync.Mutex
	isScanning   bool
	scanClient   *http.Client
)

// SSE helper to send results
func streamResultsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	lastIndex := 0

	for {
		resultsMutex.Lock()
		currentLen := len(scanResults)
		var newResults []ScanResult
		if currentLen > lastIndex {
			newResults = make([]ScanResult, currentLen-lastIndex)
			copy(newResults, scanResults[lastIndex:currentLen])
			lastIndex = currentLen
		}
		scanning := isScanning
		resultsMutex.Unlock()

		for _, res := range newResults {
			data, _ := json.Marshal(res)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}

		if len(newResults) > 0 {
			flusher.Flush()
		}

		if !scanning {
			// Send a final message indicating scan is complete
			fmt.Fprintf(w, "event: done\ndata: {}\n\n")
			flusher.Flush()
			break
		}

		time.Sleep(500 * time.Millisecond)

		// Check if client disconnected
		select {
		case <-r.Context().Done():
			return
		default:
		}
	}
}

func startScanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TargetURL   string `json:"target_url"`
		Concurrency int    `json:"concurrency"`
		Recursive   bool   `json:"recursive"`
		Wordlist    string `json:"wordlist"` // text content of wordlist
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resultsMutex.Lock()
	if isScanning {
		resultsMutex.Unlock()
		http.Error(w, "A scan is already in progress", http.StatusConflict)
		return
	}
	resultsMutex.Unlock()

	// Parse wordlist
	var words []string
	scanner := bufio.NewScanner(bytes.NewReader([]byte(req.Wordlist)))
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" && !strings.HasPrefix(word, "#") {
			words = append(words, word)
		}
	}

	if len(words) == 0 {
		// default fallback
		words = []string{"admin", "login", "api", "css", "js", "images", "config"}
	}

	config := ScanConfig{
		TargetURL:   req.TargetURL,
		Wordlist:    words,
		Concurrency: req.Concurrency,
		Recursive:   req.Recursive,
		TimeoutMs:   5000,
	}

	if config.Concurrency <= 0 {
		config.Concurrency = 10
	}

	// Reset state
	resultsMutex.Lock()
	scanResults = []ScanResult{}
	resultsMutex.Unlock()

	// Start scan in background
	// For this initial version, we use a simpler non-recursive runner to ensure it builds
	go runCrawler(config)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// runCrawler is a more robust crawler that handles recursion properly
func runCrawler(config ScanConfig) {
	resultsMutex.Lock()
	isScanning = true
	resultsMutex.Unlock()

	defer func() {
		resultsMutex.Lock()
		isScanning = false
		resultsMutex.Unlock()
	}()

	scanClient = &http.Client{
		Timeout: time.Duration(config.TimeoutMs) * time.Millisecond,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	baseURL, err := url.Parse(config.TargetURL)
	if err != nil {
		return
	}

	queue := []string{""} // start with root
	scannedPaths := make(map[string]bool)

	for len(queue) > 0 {
		currentBase := queue[0]
		queue = queue[1:]

		// Generate jobs for current base
		jobs := make(chan string, len(config.Wordlist))
		for _, word := range config.Wordlist {
			var path string
			if currentBase == "" {
				path = "/" + strings.TrimLeft(word, "/")
			} else {
				path = strings.TrimRight(currentBase, "/") + "/" + strings.TrimLeft(word, "/")
			}

			if !scannedPaths[path] {
				scannedPaths[path] = true
				jobs <- path
			}
		}
		close(jobs)

		var wg sync.WaitGroup
		var newDirectories []string
		var dirMutex sync.Mutex

		// Start workers for this batch
		numWorkers := config.Concurrency
		if numWorkers > len(config.Wordlist) {
			numWorkers = len(config.Wordlist)
		}

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for path := range jobs {
					testURL := baseURL.Scheme + "://" + baseURL.Host + path
					req, err := http.NewRequest("GET", testURL, nil)
					if err != nil {
						continue
					}
					req.Header.Set("User-Agent", "AdvancedDirScanner/1.0")

					resp, err := scanClient.Do(req)
					if err != nil {
						continue
					}

					statusCode := resp.StatusCode
					size := resp.ContentLength
					resp.Body.Close()

					if statusCode == 200 || statusCode == 204 || statusCode == 301 || statusCode == 302 || statusCode == 403 {
						res := ScanResult{
							Path:       path,
							URL:        testURL,
							StatusCode: statusCode,
							Size:       size,
						}

						resultsMutex.Lock()
						scanResults = append(scanResults, res)
						resultsMutex.Unlock()

						// If recursive and we found something that looks like a directory
						// (Usually 301 redirects to path/ or 403 on a dir, or 200)
						if config.Recursive && (statusCode == 301 || statusCode == 403 || statusCode == 200) {
							// Basic heuristic: if it doesn't have an extension, assume it might be a dir
							if !strings.Contains(path[strings.LastIndex(path, "/"):], ".") {
								dirMutex.Lock()
								newDirectories = append(newDirectories, path)
								dirMutex.Unlock()
							}
						}
					}
				}
			}()
		}
		wg.Wait()

		// Add new directories to queue
		queue = append(queue, newDirectories...)
	}
}

func main() {
	// API endpoints
	http.HandleFunc("/api/scan", startScanHandler)
	http.HandleFunc("/api/stream", streamResultsHandler)

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	port := "8080"
	log.Printf("Server listening on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
