package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func doJSON(ctx context.Context, httpClient *http.Client, op, url, method string, body, out any) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("%s: marshal request body: %w", op, err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("%s: build request: %w", op, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNodeUnreachable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return fmt.Errorf("%w: %s returned %d%s", ErrTxRejected, op, resp.StatusCode, readErrorBody(resp.Body))
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: unexpected status %d%s", op, resp.StatusCode, readErrorBody(resp.Body))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%s: decode response: %w", op, err)
	}
	return nil
}

func readErrorBody(body io.Reader) string {
	raw, _ := io.ReadAll(io.LimitReader(body, 4096))
	msg := strings.TrimSpace(string(raw))
	if msg == "" {
		return ""
	}
	return ": " + msg
}
