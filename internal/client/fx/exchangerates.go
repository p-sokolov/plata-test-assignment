package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"plata-test-assignment/internal/errorz"
)

func (c *Client) GetRate(ctx context.Context, pair string) (float64, error) {
	result, err := ConvertCurrency(ctx, c, pair, 1)
	if err != nil {
		return 0, err
	}

	return result.Result, err
}

func ConvertCurrency(
	ctx context.Context,
	c *Client,
	pair string, 
	amount float64,
) (*FXResponse, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed URL parsing: %w", err)
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
		return nil, fmt.Errorf("Request creating error: %w", err)
	}

	req.Header.Add("Accept", "application/json, application/json; Charset=UTF-8")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Response parsing failed: %w", err)
	}

	// Parse json to struct
	var apiRes FXResponse
	if err := json.Unmarshal(body, &apiRes); err != nil {
		return nil, fmt.Errorf("JSON parsing failed: %w", err)
	}

	if !apiRes.Success {
		return &apiRes, fmt.Errorf("API server error")
	}

	return &apiRes, nil
}