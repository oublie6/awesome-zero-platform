package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func runHealthcheck(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("healthcheck URL must not be empty")
	}

	client := &http.Client{Timeout: 3 * time.Second}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create healthcheck request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request liveness endpoint: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("liveness endpoint returned HTTP %d", response.StatusCode)
	}
	return nil
}
