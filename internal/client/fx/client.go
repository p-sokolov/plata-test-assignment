package fx

import (
	"net/http"
)

type Client struct {
	baseURL    string
	accessKey  string
	httpClient *http.Client
}

func New(baseURL, accessKey string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		accessKey:  accessKey,
		httpClient: httpClient,
	}
}

type FXResponse struct {
	Success bool   `json:"success"`
	Terms   string `json:"terms"`
	Privacy string `json:"privacy"`
	Query   struct {
		From   string  `json:"from"`
		To     string  `json:"to"`
		Amount float64 `json:"amount"`
	} `json:"query"`
	Info struct {
		Timestamp int64   `json:"timestamp"`
		Quote     float64 `json:"quote"`
	} `json:"info"`
	Date   string  `json:"date"`
	Result float64 `json:"result"`
}
