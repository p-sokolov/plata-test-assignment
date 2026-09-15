package quotes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"plata-test-assignment/internal/models"

	"github.com/redis/go-redis/v9"
)

const latestKeyPrefix = "quotes:latest:"

type Cache struct {
	client redis.Cmdable
	ttl    time.Duration
}

func New(client redis.Cmdable, ttl time.Duration) *Cache {
	return &Cache{client: client, ttl: ttl}
}

func (c *Cache) GetLatest(ctx context.Context, pair string) (*models.LatestQuote, error) {
	if c.client == nil {
		return nil, nil
	}
	value, err := c.client.Get(ctx, latestKey(pair)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest quote from cache failed: %w", err)
	}

	var quote models.LatestQuote
	if err := json.Unmarshal([]byte(value), &quote); err != nil {
		return nil, fmt.Errorf("decode cached latest quote failed: %w", err)
	}
	return &quote, nil
}

func (c *Cache) SetLatest(ctx context.Context, quote models.LatestQuote) error {
	if c.client == nil {
		return nil
	}
	
	value, err := json.Marshal(quote)
	if err != nil {
		return fmt.Errorf("encode latest quote for cache failed: %w", err)
	}
	if err := c.client.Set(ctx, latestKey(quote.CurrencyPair), value, c.ttl).Err(); err != nil {
		return fmt.Errorf("set latest quote in cache failed: %w", err)
	}
	return nil
}

func (c *Cache) DeleteLatest(ctx context.Context, pair string) error {
	if c.client == nil {
		return nil
	}
	if err := c.client.Del(ctx, latestKey(pair)).Err(); err != nil {
		return fmt.Errorf("delete latest quote from cache failed: %w", err)
	}
	return nil
}

func latestKey(pair string) string {
	return latestKeyPrefix + pair
}
