package redis

import (
	"github.com/go-redis/redis"
)

var c *redis.Client

func InitRedis(r *redis.Options) {
	c = redis.NewClient(r)
}
func GetRedisClient() *redis.Client {
	return c
}
