package source

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/clash-proxyd/proxyd/internal/logx"

	"go.uber.org/zap"
)

// Fetcher downloads subscription configurations
type Fetcher struct {
	client     *http.Client
	userAgent  string
	maxRetries int
	retryDelay time.Duration
}

// NewFetcher creates a new subscription fetcher
func NewFetcher(userAgent string, timeout, maxRetries, retryDelay int) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		userAgent:  userAgent,
		maxRetries: maxRetries,
		retryDelay: time.Duration(retryDelay) * time.Second,
	}
}

// Fetch fetches a subscription from a URL
func (f *Fetcher) Fetch(url string) ([]byte, error) {
	var lastErr error
	for i := 0; i < f.maxRetries; i++ {
		if i > 0 {
			logx.Warn("Retrying subscription fetch",
				zap.String("url", url),
				zap.Int("attempt", i+1))
			time.Sleep(f.retryDelay)
		}

		data, err := f.fetchOnce(url)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("failed after %d retries: %w", f.maxRetries, lastErr)
}

// fetchOnce performs a single fetch attempt
func (f *Fetcher) fetchOnce(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	logx.Info("Subscription fetched successfully",
		zap.String("url", url),
		zap.Int("size", len(data)))

	return data, nil
}

// FetchFromFile reads a subscription from a local file
func (f *Fetcher) FetchFromFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	logx.Info("Subscription read from file",
		zap.String("path", path),
		zap.Int("size", len(data)))

	return data, nil
}

// Hash calculates the SHA256 hash of data
func Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Save saves data to a file
func Save(data []byte, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// TestSubscription tests a subscription URL
func TestSubscription(url string, timeout int) (bool, int, error) {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, 0, err
	}

	req.Header.Set("User-Agent", "clash-proxyd/test")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	latency := int(time.Since(start).Milliseconds())

	if resp.StatusCode != http.StatusOK {
		return false, latency, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return true, latency, nil
}
