package secretmessage

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Handle secret expiration that flush secrets in DB
func FlushExpiredSecrets(ctl *PublicController, secret *Secret) error {
	for {
		now := time.Now()

		if now.After(secret.ExpiresAt) || now.Equal(secret.ExpiresAt) {
			if err := ctl.db.WithContext(context.TODO()).Unscoped().Where("id = ?", secret.ID).Delete(Secret{}).Error; err != nil {
				ctl.logger.Error("could not flush expired secret", zap.Error(err), zap.String("secret_id", secret.ID))
			}
			ctl.logger.Info("secret successfully flushed", zap.String("secret_id", secret.ID))
			break
		}

		time.Sleep(time.Second * 300)
	}

	return nil
}
