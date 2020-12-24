module github.com/neufeldtech/secretmessage-go

// +heroku goVersion go1.13
go 1.13

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/alicebob/miniredis/v2 v2.11.4
	github.com/gin-gonic/gin v1.5.0
	github.com/go-redis/redis v6.15.7+incompatible
	github.com/golang-migrate/migrate/v4 v4.14.1
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/jackc/pgx/v4 v4.10.1 // indirect
	github.com/jarcoal/httpmock v1.0.6
	github.com/kr/pretty v0.2.0 // indirect
	github.com/lib/pq v1.9.0
	github.com/lithammer/shortuuid v3.0.0+incompatible
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/common v0.9.1
	github.com/slack-go/slack v0.6.4
	github.com/stretchr/testify v1.6.1
	github.com/tj/assert v0.0.3
	go.elastic.co/apm v1.9.0
	go.elastic.co/apm/module/apmgin v1.8.0
	go.elastic.co/apm/module/apmgoredis v1.8.0
	go.elastic.co/apm/module/apmhttp v1.8.0
	go.elastic.co/apm/module/apmsql v1.9.0
	golang.org/x/crypto v0.0.0-20201217014255-9d1352758620 // indirect
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/text v0.3.4 // indirect
	golang.org/x/tools v0.0.0-20201221201019-196535612888 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200605160147-a5ece683394c // indirect
	gorm.io/driver/postgres v1.0.5
	gorm.io/gorm v1.20.8
)
