package secretmessage

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/prometheus/common/log"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgoredis"
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

func MigrateSecretsToPostgres(redis *redis.Client, db *gorm.DB) error {
	c := context.Background()
	tx := apm.DefaultTracer.StartTransaction("Migrate", "background_job")
	defer tx.End()
	ctx := apm.ContextWithTransaction(c, tx)
	rc := apmgoredis.Wrap(redis)

	var teamKeys []string
	var secretKeys []string
	var teamsMigrated int
	var secretsMigrated int

	log.Info("Attempting to migrate...")

	d, _ := db.DB()
	err := d.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("error pinging db %v", err)
	}

	keys, err := rc.WithContext(ctx).Keys("*").Result()
	if err != nil {
		return fmt.Errorf("error getting keys from redis %v", keys)
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

	for _, teamID := range teamKeys {
		team, err := rc.WithContext(ctx).HMGet(teamID, "name", "access_token", "scope").Result()
		if err != nil {
			log.Errorf("error getting key %v from redis %v\n", teamID, err)
			continue
		}
		teamName := team[0].(string)
		accessToken := team[1].(string)
		scope := team[2].(string)

		err = db.WithContext(ctx).Create(
			&Team{
				ID:          teamID,
				Name:        teamName,
				AccessToken: accessToken,
				Scope:       scope,
			},
		).Error
		if err != nil {
			log.Errorf("error inserting team %v into db %v", teamID, err)
			continue
		}
		teamsMigrated++
	}

	for _, secretID := range secretKeys {
		secretValue, err := rc.WithContext(ctx).Get(secretID).Result()
		if err != nil {
			log.Errorf("error getting key %v from redis %v\n", secretID, err)
			continue
		}
		err = db.WithContext(ctx).Create(
			&Secret{
				ID:        secretID,
				ExpiresAt: time.Now().Add(time.Hour * 300),
				Value:     secretValue,
			},
		).Error
		if err != nil {
			log.Errorf("error inserting secret %v into db %v", secretID, err)
			continue
		}
		secretsMigrated++
	}

	log.Infof("successfully migrated %v teams and %v secrets", teamsMigrated, secretsMigrated)
	return nil
}
