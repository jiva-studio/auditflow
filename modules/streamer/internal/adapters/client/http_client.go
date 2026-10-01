// Package client implements the outbound HTTP client adapter for sending tick batches to the agent.
package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"assessment/libs/domain/events"
	"assessment/modules/streamer/internal/adapters/mapper"
	"assessment/modules/streamer/internal/ports"
)

// Sentinel errors for HTTPClient.
var (
	ErrEmptyAgentURL    = errors.New("agent URL cannot be empty")
	ErrNonSuccessStatus = errors.New("agent returned non-success HTTP status")
)

// HTTPClient dispatches tick batches over HTTP to an agent service.
type HTTPClient struct {
	agentURL   string
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTPClient pointing to the target agent URL.
func NewHTTPClient(agentURL string, timeout time.Duration) (*HTTPClient, error) {
	if strings.TrimSpace(agentURL) == "" {
		return nil, ErrEmptyAgentURL
	}
	baseURL := strings.TrimRight(agentURL, "/")
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	return &HTTPClient{
		agentURL: baseURL,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
	}, nil
}

var _ ports.AgentClient = (*HTTPClient)(nil)

// SendTick converts a domain TickBatch to protobuf via the mapper and dispatches it via POST /tick.
func (c *HTTPClient) SendTick(ctx context.Context, batch events.TickBatch) error {
	pbBatch := mapper.ToProtoTickBatch(batch)
	data, err := proto.Marshal(pbBatch)
	if err != nil {
		return fmt.Errorf("marshal protobuf tick batch: %w", err)
	}

	reqURL := c.agentURL + "/tick"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dispatch tick HTTP request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: %d", ErrNonSuccessStatus, resp.StatusCode)
	}

	return nil
}
