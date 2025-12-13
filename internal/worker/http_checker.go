package worker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/shubhbham/uptime-monitor/internal/domain"
	"github.com/shubhbham/uptime-monitor/internal/httpclient"
)

type HTTPChecker struct {
	client *httpclient.Client
}

func NewHTTPChecker(client *httpclient.Client) *HTTPChecker {
	return &HTTPChecker{
		client: client,
	}
}

func (h *HTTPChecker) Check(ctx context.Context, monitor *domain.Monitor) *domain.CheckResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, monitor.Method, monitor.URL, nil)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create request: %v", err)
		return &domain.CheckResult{
			IsUp:         false,
			ErrorMessage: &errMsg,
		}
	}

	req.Header.Set("User-Agent", "UptimeMonitor/1.0")

	h.client.SetTimeout(time.Duration(monitor.TimeoutSeconds) * time.Second)

	resp, err := h.client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("Request failed: %v", err)
		responseTime := int(time.Since(start).Milliseconds())
		return &domain.CheckResult{
			ResponseTimeMs: &responseTime,
			IsUp:           false,
			ErrorMessage:   &errMsg,
		}
	}
	defer resp.Body.Close()

	responseTime := int(time.Since(start).Milliseconds())
	statusCode := resp.StatusCode

	isUp := statusCode == monitor.ExpectedStatus

	result := &domain.CheckResult{
		StatusCode:     &statusCode,
		ResponseTimeMs: &responseTime,
		IsUp:           isUp,
	}

	if !isUp {
		errMsg := fmt.Sprintf("Unexpected status code: %d (expected %d)", statusCode, monitor.ExpectedStatus)
		result.ErrorMessage = &errMsg
	}

	return result
}