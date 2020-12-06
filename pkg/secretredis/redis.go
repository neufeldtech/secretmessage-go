package secretredis

import (
	"github.com/go-redis/redis"
	"go.elastic.co/apm/module/apmgoredis"
)

var c *redis.Client

func Connect(r *redis.Options) {
	c = redis.NewClient(r)
}

func Client() apmgoredis.Client {
	return apmgoredis.Wrap(c)
}
