package secretredis

import (
	"github.com/go-redis/redis"
)

var c *redis.Client

func Connect(r *redis.Options) {
	c = redis.NewClient(r)
}
func Client() *redis.Client {
	return c
}
