package monitor

import (
	"errors"
	"net/url"
	"strings"

	"github.com/shubhbham/uptime-monitor/internal/domain"
)

var (
	ErrInvalidName            = errors.New("name is required and must be between 1-255 characters")
	ErrInvalidURL             = errors.New("invalid URL format")
	ErrInvalidMethod          = errors.New("method must be GET, POST, PUT, DELETE, PATCH, or HEAD")
	ErrInvalidExpectedStatus  = errors.New("expected status must be between 100-599")
	ErrInvalidInterval        = errors.New("interval must be between 30-3600 seconds")
	ErrInvalidTimeout         = errors.New("timeout must be between 1-120 seconds")
)

func ValidateCreateRequest(req *domain.CreateMonitorRequest) error {
	if strings.TrimSpace(req.Name) == "" || len(req.Name) > 255 {
		return ErrInvalidName
	}

	if _, err := url.ParseRequestURI(req.URL); err != nil {
		return ErrInvalidURL
	}

	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true,
		"DELETE": true, "PATCH": true, "HEAD": true,
	}
	if !validMethods[strings.ToUpper(req.Method)] {
		return ErrInvalidMethod
	}

	if req.ExpectedStatus < 100 || req.ExpectedStatus > 599 {
		return ErrInvalidExpectedStatus
	}

	if req.IntervalSeconds < 30 || req.IntervalSeconds > 3600 {
		return ErrInvalidInterval
	}

	if req.TimeoutSeconds < 1 || req.TimeoutSeconds > 120 {
		return ErrInvalidTimeout
	}

	return nil
}

func ValidateUpdateRequest(req *domain.UpdateMonitorRequest) error {
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" || len(*req.Name) > 255 {
			return ErrInvalidName
		}
	}

	if req.URL != nil {
		if _, err := url.ParseRequestURI(*req.URL); err != nil {
			return ErrInvalidURL
		}
	}

	if req.Method != nil {
		validMethods := map[string]bool{
			"GET": true, "POST": true, "PUT": true,
			"DELETE": true, "PATCH": true, "HEAD": true,
		}
		if !validMethods[strings.ToUpper(*req.Method)] {
			return ErrInvalidMethod
		}
	}

	if req.ExpectedStatus != nil {
		if *req.ExpectedStatus < 100 || *req.ExpectedStatus > 599 {
			return ErrInvalidExpectedStatus
		}
	}

	if req.IntervalSeconds != nil {
		if *req.IntervalSeconds < 30 || *req.IntervalSeconds > 3600 {
			return ErrInvalidInterval
		}
	}

	if req.TimeoutSeconds != nil {
		if *req.TimeoutSeconds < 1 || *req.TimeoutSeconds > 120 {
			return ErrInvalidTimeout
		}
	}

	return nil
}