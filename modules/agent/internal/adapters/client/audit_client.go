// Package client implements the outbound HTTP client adapter for sending audit popups to the central server.
package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"assessment/libs/domain/audit"
	"assessment/modules/agent/internal/adapters/mapper"
	"assessment/modules/agent/internal/ports"
)

// Sentinel errors for AuditHTTPClient.
var (
	ErrEmptyServerURL   = errors.New("server URL cannot be empty")
	ErrNonSuccessStatus = errors.New("server returned non-success HTTP status")
)

// AuditHTTPClient dispatches audit popups over HTTP to the central audit server.
type AuditHTTPClient struct {
	serverURL  string
	httpClient *http.Client
}

// NewAuditHTTPClient creates a new AuditHTTPClient with the given base URL and timeout.
func NewAuditHTTPClient(serverURL string, timeout time.Duration) (*AuditHTTPClient, error) {
	if strings.TrimSpace(serverURL) == "" {
		return nil, ErrEmptyServerURL
	}
	baseURL := strings.TrimRight(serverURL, "/")
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	return &AuditHTTPClient{
		serverURL: baseURL,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
	}, nil
}

var _ ports.AuditClient = (*AuditHTTPClient)(nil)

// SendPopup encodes a domain Popup to protobuf and sends it via POST /audit.
func (c *AuditHTTPClient) SendPopup(ctx context.Context, popup audit.Popup) error {
	pbPopup := mapper.ToProtoPopup(popup)
	data, err := proto.Marshal(pbPopup)
	if err != nil {
		return fmt.Errorf("marshal protobuf popup: %w", err)
	}

	reqURL := c.serverURL + "/audit"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dispatch popup HTTP request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: %d", ErrNonSuccessStatus, resp.StatusCode)
	}

	return nil
}
