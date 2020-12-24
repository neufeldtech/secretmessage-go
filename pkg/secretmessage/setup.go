package secretmessage

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/prometheus/common/log"
	"gorm.io/gorm"
)

func callHealth(url string) error {
	resp, err := http.Get(url + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func StayAwake(config Config) {
	for {
		time.Sleep(5 * time.Minute)
		err := callHealth(config.AppURL)
		if err != nil {
			log.Error(err)
		}
	}
}

func MigrateSecretsToPostgres(redis *redis.Client, db *gorm.DB) {
	var teamKeys []string
	var secretKeys []string

	log.Info("Attempting to migrate...")
	keys, err := redis.Keys("*").Result()
	if err != nil {
		log.Fatalf("error getting keys from redis %v", keys)
	}
	log.Infof("found %v keys ....", len(keys))

	for _, k := range keys {
		if len(k) == 64 {
			secretKeys = append(secretKeys, k)
			log.Infof("found secret key %v", k)
			continue
		}
		if len(k) < 64 && strings.HasPrefix(k, "T") {
			teamKeys = append(teamKeys, k)
			log.Infof("found team key %v", k)
			continue
		}
	}

}
