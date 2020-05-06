package secretmessage

import (
	"github.com/go-redis/redis"
)

var c *redis.Client

func InitRedis(config Config) {
	c = redis.NewClient(config.RedisOptions)
}
func GetRedisClient() *redis.Client {
	return c
}
