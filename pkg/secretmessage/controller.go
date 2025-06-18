package secretmessage

import (
	"net/http"
	"os"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/neufeldtech/secretmessage-go/pkg/secretslack"
	"go.elastic.co/apm/module/apmgin"
	"go.elastic.co/apm/module/apmhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PublicController struct {
	db           *gorm.DB
	config       Config
	logger       *zap.Logger
	slackService *secretslack.SlackService
}

func NewController(config Config, db *gorm.DB, logger *zap.Logger) *PublicController {
	if logger == nil {
		logger = zap.Must(zap.NewProduction())
	}

	slackService := secretslack.NewSlackService().
		WithHTTPClient(
			apmhttp.WrapClient(
				&http.Client{
					Timeout: 5 * time.Second},
			)).
		WithLogger(logger)

	return &PublicController{
		db:           db,
		config:       config,
		logger:       logger,
		slackService: slackService,
	}
}

func (ctl *PublicController) ConfigureRoutes() *gin.Engine {

	r := gin.New()
	r.Use(ginzap.Ginzap(ctl.logger, time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(ctl.logger, true))

	r.Use(otelgin.Middleware(os.Getenv("HOSTNAME")))
	r.Use(apmgin.Middleware(r))
	r.Use(func(c *gin.Context) {
		c.Next()
		db, err := ctl.db.DB()
		if err == nil {
			stats := db.Stats()
			ctl.logger.Info("DB Stats",
				zap.Int("OpenConnections", stats.OpenConnections),
				zap.Int("InUse", stats.InUse),
				zap.Int("Idle", stats.Idle),
				zap.Int64("WaitCount", stats.WaitCount),
				zap.Duration("WaitDuration", stats.WaitDuration),
				zap.Int("MaxOpenConnections", stats.MaxOpenConnections),
				zap.Int64("MaxIdleClosed", stats.MaxIdleClosed),
				zap.Int64("MaxIdleTimeClosed", stats.MaxIdleTimeClosed),
				zap.Int64("MaxLifetimeClosed", stats.MaxLifetimeClosed),
			)
		}
	})
	r.GET("/health", ctl.HandleHealth)

	r.GET("/auth/slack", ctl.HandleOauthBegin)
	r.GET("/auth/slack/callback", ctl.HandleOauthCallback)

	// Signature validation required
	r.POST("/slash", ctl.ValidateSignature(), ctl.HandleSlash)
	r.POST("/interactive", ctl.ValidateSignature(), ctl.HandleInteractive)

	return r
}
