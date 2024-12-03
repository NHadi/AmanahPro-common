package persistence

import (
	"context"

	"github.com/go-redis/redis/v8"
)

// InitializeRedis initializes a Redis client with options for password and DB selection.
func InitializeRedis(redisURL string, password string, db int) (*redis.Client, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		// Fallback if redisURL is not in standard format
		options = &redis.Options{
			Addr:     redisURL,
			Password: password, // Use the provided password
			DB:       db,       // Use the provided DB number
		}
	} else {
		// Update password and DB if they are explicitly provided
		if password != "" {
			options.Password = password
		}
		options.DB = db
	}

	client := redis.NewClient(options)

	// Test the connection
	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	return client, nil
}
