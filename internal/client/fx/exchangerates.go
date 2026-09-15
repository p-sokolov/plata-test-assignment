package fx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"plata-test-assignment/internal/errorz"
)

func (c *Client) GetRate(ctx context.Context, pair string) (float64, error) {
	result, err := c.convertCurrency(ctx, pair, 1)
	if err != nil {
		return 0, err
	}

	return result.Result, err
}

func (c *Client) convertCurrency(
	ctx context.Context,
	pair string,
	amount float64,
) (response *FXResponse, err error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	if len(pair) != 7 {
		return nil, errorz.ErrUnsupportedCurrency
	}
	from := pair[:3]
	to := pair[4:]

	// Parse query params
	q := u.Query()
	q.Add("access_key", c.accessKey)
	q.Add("from", from)
	q.Add("to", to)
	q.Add("amount", fmt.Sprintf("%v", amount))
	u.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Add("Accept", "application/json, application/json; Charset=UTF-8")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close response body: %w", closeErr))
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("response parsing failed: %w", err)
	}

	// All status codes described in exchanger specs
	switch res.StatusCode {
	case http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable:
		return nil, fmt.Errorf("client %s with status code: %d", http.StatusText(res.StatusCode), res.StatusCode)
	default:
		if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("unexpected response status: %s", res.Status)
		}
	}

	// Parse json to struct
	var apiRes FXResponse
	if err := json.Unmarshal(body, &apiRes); err != nil {
		return nil, fmt.Errorf("parse JSON response: %w", err)
	}

	if !apiRes.Success {
		return &apiRes, errors.New("api server error")
	}

	return &apiRes, nil
}
