package secretmessage

import (
	"github.com/go-redis/redis"
)

var c *redis.Client

func InitRedis(config Config) {
	c = redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
		DB:       0, // use default DB
	})
}
func GetRedisClient() *redis.Client {
	return c
}
