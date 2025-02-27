// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/time/rate"
)

type m3terClient struct {
	organizationID string
	client         *http.Client
	limit          *rate.Limiter
}

func (c *m3terClient) execute(ctx context.Context, method string, path string, query url.Values, requestBody any, responseBody any) error {
	err := c.limit.Wait(ctx)
	if err != nil {
		return err
	}
	fullURL := "https://api.m3ter.com/organizations/" + url.PathEscape(c.organizationID) + path
	if query != nil {
		fullURL += "?" + query.Encode()
	}

	var requestBodyReader io.Reader
	if requestBody != nil {
		body, err := json.Marshal(requestBody)
		if err != nil {
			return err
		}
		requestBodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, requestBodyReader)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return &statusCodeError{StatusCode: resp.StatusCode}
		}
		return &statusCodeError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	if responseBody != nil {
		err = json.NewDecoder(resp.Body).Decode(responseBody)
		if err != nil {
			return err
		}
	}
	return nil
}

type statusCodeError struct {
	StatusCode int
	Body       string
}

func (e *statusCodeError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("unexpected status code %d", e.StatusCode)
	}
	return fmt.Sprintf("unexpected status code %d: %s", e.StatusCode, e.Body)
}
