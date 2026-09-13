package errorz

import "errors"

var (
	ErrUnsupportedCurrency = errors.New("unsupported currency pair")
	ErrQuoteNotFound = errors.New("quote not found")
)