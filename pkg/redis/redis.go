package redis

import (
	"github.com/go-redis/redis"
)

var c *redis.Client

func Init() {
	c = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}
func GetClient() *redis.Client {
	return c
}
