// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/cenkalti/backoff/v4"

	"errors"
	"net/http"
	"time"
)

var errClientRateLimitExceeded = errors.New("m3ter rate limit exceeded")

type BackoffRateLimiter struct {
	backoffProvider func() backoff.BackOff
}

func NewBackoffRateLimiter() *BackoffRateLimiter {
	return &BackoffRateLimiter{backoffProvider: func() backoff.BackOff {
		return &backoff.ExponentialBackOff{
			InitialInterval:     3 * time.Second / 4,
			RandomizationFactor: 0.25,
			Multiplier:          2,
			MaxInterval:         10 * time.Second,
			MaxElapsedTime:      20 * time.Second,
			Clock:               backoff.SystemClock,
			Stop:                backoff.Stop,
		}
	}}
}

func (r *BackoffRateLimiter) doWithLimit(f func() (*http.Response, error)) (*http.Response, error) {
	return backoff.RetryWithData(func() (*http.Response, error) {
		resp, err := f()
		if err != nil {
			return nil, backoff.Permanent(err)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, errClientRateLimitExceeded
		}
		return resp, nil
	}, r.backoffProvider())
}
